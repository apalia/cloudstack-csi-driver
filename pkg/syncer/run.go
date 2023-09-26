package syncer

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/apalia/cloudstack-csi-driver/pkg/driver"
)

var (
	volBindingMode       = storagev1.VolumeBindingWaitForFirstConsumer
	reclaimPolicy        = corev1.PersistentVolumeReclaimDelete
	allowVolumeExpansion = false
)

func (s syncer) Run(ctx context.Context) error {
	oldSc := make([]string, 0)
	newSc := make([]string, 0)
	errs := make([]error, 0)

	// List existing K8s storage classes

	labelSelector := s.labelsSet.String()
	log.Printf("Listing Storage classes with label selector \"%s\"...", labelSelector)
	scList, err := s.k8sClient.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("cannot list existing storage classes: %w", err)
	}
	for _, sc := range scList.Items {
		oldSc = append(oldSc, sc.Name)
	}
	log.Printf("Found %v: %v\n", len(oldSc), oldSc)

	// List CloudStack disk offerings

	log.Println("Listing CloudStack disk offerings...")
	p := s.csClient.DiskOffering.NewListDiskOfferingsParams()
	diskOfferings, err := s.csClient.DiskOffering.ListDiskOfferings(p)
	if err != nil {
		return fmt.Errorf("cannot list CloudStack disk offerings: %w", err)
	}

	// Iterate over CloudStack disk offerings to synchronize them

	for _, offering := range diskOfferings.DiskOfferings {
		name, err := s.syncOffering(ctx, offering)
		if err != nil {
			err = fmt.Errorf("Error with offering %s: %w", offering.Name, err)
			log.Println(err.Error())
			errs = append(errs, err)
		}
		if name != "" {
			newSc = append(newSc, name)
		}
	}
	log.Println("No more CloudStack disk offerings")

	// If enabled, delete unused labeled storage classes

	if s.delete {
		del := toDelete(oldSc, newSc)
		if len(del) == 0 {
			log.Println("No storage class to delete")
		} else {
			for _, sc := range del {
				log.Printf("Deleting storage class %s", sc)
				err = s.k8sClient.StorageV1().StorageClasses().Delete(ctx, sc, metav1.DeleteOptions{})
				if err != nil {
					err = fmt.Errorf("error deleting storage class %s: %w", sc, err)
					log.Println(err.Error())
					errs = append(errs, err)
				}
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return combinedError(errs)
}

func (s syncer) syncOffering(ctx context.Context, offering *cloudstack.DiskOffering) (string, error) {
	offeringName := offering.Name
	custom := offering.Iscustomized
	if !custom {
		log.Printf("Disk offering \"%s\" has a fixed size: ignoring\n", offeringName)
		return "", nil
	}

	log.Printf("Syncing disk offering %s...", offeringName)
	name, err := createStorageClassName(s.namePrefix + offeringName)
	if err != nil {
		log.Printf("Cannot transform name: %s", err.Error())
		name = offering.Id
	}
	log.Printf("Storage class name: %s", name)

	sc, err := s.k8sClient.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {

			// Storage class does not exist; creating it

			log.Printf("Creating storage class %s", name)

			newSc := &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:   name,
					Labels: s.labelsSet,
				},
				Provisioner:          driver.DriverName,
				VolumeBindingMode:    &volBindingMode,
				ReclaimPolicy:        &reclaimPolicy,
				AllowVolumeExpansion: &allowVolumeExpansion,
				Parameters: map[string]string{
					driver.DiskOfferingKey: offering.Id,
				},
			}

			//Add AllowedTopologies if the addAllowedTopology flag is true
			if s.addAllowedTopology {
				var zoneID string
				vm, err := s.csConnector.GetNodeInfo(ctx, s.nodeName)
				if err != nil {
					log.Printf("GetNodeinfo failed: %s", err.Error())
				} else {
					zoneID = vm.ZoneID
				}

				allowedtopology := []corev1.TopologySelectorTerm{
					{
						MatchLabelExpressions: []corev1.TopologySelectorLabelRequirement{
							{
								Key:    "topology." + driver.DriverName + "/zone",
								Values: []string{zoneID},
							},
						},
					},
				}
				newSc.AllowedTopologies = allowedtopology
			}

			_, err = s.k8sClient.StorageV1().StorageClasses().Create(ctx, newSc, metav1.CreateOptions{})
			return name, err
		}
		return "", err
	}

	// Storage class already exists

	err = checkStorageClass(sc, offering.Id)
	if err != nil {
		// Updates to provisioner, reclaimpolicy, volumeBindingMode and parameters are forbidden
		log.Printf("Storage class %s exists but it not compatible.", name)
		return name, err
	}

	// Update labels if needed

	existingLabels := labels.Set(sc.Labels)
	if !s.labelsSet.AsSelector().Matches(existingLabels) {
		log.Printf("Storage class %s misses labels %s: updating...", sc.Name, s.labelsSet.String())

		sc.Labels = labels.Merge(existingLabels, s.labelsSet)
		_, err = s.k8sClient.StorageV1().StorageClasses().Update(ctx, sc, metav1.UpdateOptions{})
		return name, err
	}

	log.Printf("Storage class %s already ok", sc.Name)

	return name, nil
}

func checkStorageClass(sc *storagev1.StorageClass, expectedOfferingID string) error {
	errs := make([]error, 0)
	diskOfferingID, ok := sc.Parameters[driver.DiskOfferingKey]
	if !ok {
		errs = append(errs, fmt.Errorf("missing parameter %s", driver.DiskOfferingKey))
	} else if diskOfferingID != expectedOfferingID {
		errs = append(errs, fmt.Errorf("storage class %s has parameter %s=%s, should be %s", sc.Name, driver.DiskOfferingKey, diskOfferingID, expectedOfferingID))
	}

	if sc.ReclaimPolicy == nil || *sc.ReclaimPolicy != reclaimPolicy {
		errs = append(errs, errors.New("wrong ReclaimPolicy"))
	}
	if sc.VolumeBindingMode == nil || *sc.VolumeBindingMode != volBindingMode {
		errs = append(errs, errors.New("wrong VolumeBindingMode"))
	}
	if sc.AllowVolumeExpansion == nil || *sc.AllowVolumeExpansion != allowVolumeExpansion {
		errs = append(errs, errors.New("wrong AllowVolumeExpansion"))
	}

	if len(errs) > 0 {
		return combinedError(errs)
	}
	return nil
}

func toDelete(oldSc, newSc []string) []string {
	del := make([]string, 0)
	for _, old := range oldSc {
		var found bool
		for _, new := range newSc {
			if new == old {
				found = true
				break
			}
		}
		if !found {
			del = append(del, old)
		}
	}
	return del
}

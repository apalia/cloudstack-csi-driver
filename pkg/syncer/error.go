package syncer

import "fmt"

type combinedError []error

func (errs combinedError) Error() string {
	err := "Collected errors:\n"
	for i, e := range errs {
		err += fmt.Sprintf("\tError %d: %s\n", i, e.Error())
	}
	return err
}

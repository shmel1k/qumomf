package config

import "fmt"

func validateElector(v *string) error {
	if v == nil {
		return fmt.Errorf("option 'elector' must not be empty")
	}

	if *v != "idle" && *v != "smart" {
		return fmt.Errorf("option 'elector' has a wrong value:: %s", *v)
	}

	return nil
}

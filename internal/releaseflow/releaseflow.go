package releaseflow

import "fmt"

type Stage struct {
	Name string
	Run  func() error
}

func Run(stages []Stage) error {
	for _, stage := range stages {
		if err := stage.Run(); err != nil {
			return fmt.Errorf("%s: %w", stage.Name, err)
		}
	}

	return nil
}

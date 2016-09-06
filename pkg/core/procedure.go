package core

import "github.com/supergiant/supergiant/pkg/model"

type Procedure struct {
	Core  *Core
	Name  string
	Model model.Model
	steps []*Step
}

type Step struct {
	desc string
	fn   func() error
}

func (p *Procedure) AddStep(desc string, fn func() error) {
	p.steps = append(p.steps, &Step{desc, fn})
}

func (p *Procedure) Run() error {
	for _, step := range p.steps {
		p.Core.Log.Infof("Running step of %s procedure: %s", p.Name, step.desc)
		if err := step.fn(); err != nil {
			return err
		}
		if err := p.Core.DB.Save(p.Model); err != nil {
			return err
		}
	}
	return nil
}

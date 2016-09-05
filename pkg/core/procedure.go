package core

import "github.com/supergiant/supergiant/pkg/model"

type Procedure struct {
	core  *Core
	name  string
	model model.Model
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
		p.core.Log.Infof("Running step of %s procedure: %s", p.name, step.desc)
		if err := step.fn(); err != nil {
			return err
		}
		if err := p.core.DB.Save(p.model); err != nil {
			return err
		}
	}
	return nil
}

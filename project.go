package main

import "log"

func (p *Project) AddError() error {
	p.ErrorsCount++
	return nil
}

func (p *Project) DecError() error {
	if p.ErrorsCount > 0 {
		p.ErrorsCount--
	}
	return nil

}

func (p *Project) AddFail() error {
	p.FailsCount++
	return nil
}

func (p *Project) DecFail() error {
	if p.FailsCount > 0 {
		p.FailsCount--
	}
	return nil

}

func (p *Project) CeaseAlerts() error {
	log.Printf("Old mode: %s", p.Parameters.Mode)
	p.Parameters.Mode = "quiet"
	log.Printf("New mode: %s", p.Parameters.Mode)
	return nil
}

func (p *Project) EnableAlerts() error {
	log.Printf("Old mode: %s", p.Parameters.Mode)
	p.Parameters.Mode = "loud"
	log.Printf("New mode: %s", p.Parameters.Mode)
	return nil
}

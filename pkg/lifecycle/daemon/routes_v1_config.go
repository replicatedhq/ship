package daemon

func (d *V1Routes) MessageConfirmedChan() chan string {
	return d.MessageConfirmed
}

func (d *V1Routes) ConfigSavedChan() chan interface{} {
	return d.ConfigSaved
}

func (d *V1Routes) GetCurrentConfig() map[string]interface{} {
	if d.CurrentConfig == nil {
		return make(map[string]interface{})
	}
	return d.CurrentConfig
}

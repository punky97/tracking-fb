package mgo_mock

type Pipe struct {
	Session *Session
}
func (pipe *Pipe) All(result interface{}) error {
	return nil
}
func (pipe *Pipe) One(result interface{}) error {
	return nil
}
func (pipe *Pipe) Explain(result interface{}) error {
	return nil
}
func (p *Pipe) Batch(result interface{}) *Pipe {
	return p
}

func (p *Pipe) Iter() *Iter {
	return &Iter{}
}
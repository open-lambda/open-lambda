package boss

type Scaling interface {
	Launch(b *Boss) //launch auto-scaler
	Scale() //makes scaling decision based on cluster status
	Close() //close auto-scaler
}

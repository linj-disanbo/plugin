package executor

type DBprefix interface {
	GetLocaldbPrefix() string
	GetStatedbPrefix() string
}

const (
	//KeyPrefixStateDB state db key必须前缀
	KeyPrefixStateDB = "mavl-zkspot-"
	//KeyPrefixLocalDB local db的key必须前缀
	KeyPrefixLocalDB = "LODB-zkspot"
)

type dbprefix struct {
	//local, state string
}

func (d *dbprefix) GetLocaldbPrefix() string {
	return KeyPrefixLocalDB
}

func (d *dbprefix) GetStatedbPrefix() string {
	return KeyPrefixStateDB
}

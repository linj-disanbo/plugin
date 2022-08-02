package executor

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

type zkHandler struct {
	info *TreeUpdateInfo
}

func newZkHandler(info *TreeUpdateInfo) *zkHandler {
	return &zkHandler{
		info: info,
	}
}

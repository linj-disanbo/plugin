package spot

import (
	"encoding/json"

	dbm "github.com/33cn/chain33/common/db"

	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

func loadSpotFeeAccountConfig(db dbm.KV) (*et.DexAccount, error) {
	key := string(spotFeeAccountKey)
	value, err := getManageKey(key, db)
	if err != nil {
		elog.Info("loadSpotFeeAccountConfig", "get db key", "not found", "key", key)
		return nil, err
	}
	if value == nil {
		elog.Info("loadSpotFeeAccountConfig", "get db key", "  found nil value", "key", key)
		return nil, nil
	}
	elog.Info("loadSpotFeeAccountConfig", "value", string(value))

	var item types.ConfigItem
	err = types.Decode(value, &item)
	if err != nil {
		elog.Error("loadSpotFeeAccountConfig", "Can't decode ConfigItem due to", err.Error())
		return nil, err // types.ErrBadConfigValue
	}

	configValue := item.GetArr().Value
	if len(configValue) <= 0 {
		return nil, et.ErrSpotFeeConfig
	}

	var e et.DexAccount
	err = json.Unmarshal([]byte(configValue[0]), &e)

	if err != nil {
		elog.Error("loadSpotFeeAccountConfig load", "Can't decode token info due to:", err.Error())
		return nil, err
	}
	return &e, nil
}

func getManageKey(key string, db dbm.KV) ([]byte, error) {
	manageKey := types.ManageKey(key)
	value, err := db.Get([]byte(manageKey))
	if err != nil {
		elog.Info("getManageKey", "get db key", "not found manageKey", "key", manageKey)
		return getConfigKey(key, db)
	}
	return value, nil
}

func getConfigKey(key string, db dbm.KV) ([]byte, error) {
	configKey := types.ConfigKey(key)
	value, err := db.Get([]byte(configKey))
	if err != nil {
		elog.Info("getManageKey", "get db key", "not found configKey", "key", configKey)
		return nil, err
	}
	return value, nil
}

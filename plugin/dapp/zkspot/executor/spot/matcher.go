package spot

import (
	"github.com/33cn/chain33/client"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// market depth:
// price - list
// order - list for each price
type matcher struct {
	localdb dbm.KV
	statedb dbm.KV
	api     client.QueueProtocolAPI

	matchCount int
	maxMatch   int
	done       bool

	// price list
	pricekey     string
	endPriceList bool

	// order list
	lastOrderPrice int64
	orderKey       string
	endOrderList   bool
}

func newMatcher(statedb, localdb dbm.KV, api client.QueueProtocolAPI) *matcher {
	return &matcher{
		localdb: localdb,
		statedb: statedb,
		api:     api,

		pricekey:     "",
		matchCount:   0,
		maxMatch:     et.MaxMatchCount,
		done:         false,
		endPriceList: false,
	}
}
func (m *matcher) isDone() bool {
	return (m.done || m.matchCount >= m.maxMatch)
}

func (m *matcher) recordMatchCount() {
	m.matchCount = m.matchCount + 1
	if m.matchCount >= m.maxMatch {
		m.done = true
	}
}

func (m *matcher) priceDone(payload *et.SpotLimitOrder, marketDepth *et.SpotMarketDepth) bool {
	if priceDone(payload, marketDepth) {
		m.done = true
		return true
	}
	return false
}

func priceDone(payload *et.SpotLimitOrder, marketDepth *et.SpotMarketDepth) bool {
	if payload.Op == et.OpBuy && marketDepth.Price > payload.GetPrice() {
		return true
	}
	if payload.Op == et.OpSell && marketDepth.Price < payload.GetPrice() {
		return true
	}
	return false
}

func (m *matcher) QueryMarketDepth(payload *et.SpotLimitOrder) (*et.SpotMarketDepthList, error) {
	if m.endPriceList {
		m.done = true
		return nil, nil
	}
	marketDepthList, _ := QueryMarketDepth(m.localdb, payload.GetLeftAsset(), payload.GetRightAsset(), OpSwap(payload.Op), m.pricekey, et.Count)
	if marketDepthList == nil || len(marketDepthList.List) == 0 {
		return nil, nil
	}

	// reatch the last price list
	if marketDepthList.PrimaryKey == "" {
		m.endPriceList = true
	}

	// set next key
	m.pricekey = marketDepthList.PrimaryKey
	return marketDepthList, nil
}

func (m *matcher) findOrderIDListByPrice(payload *et.SpotLimitOrder, marketDepth *et.SpotMarketDepth) (*et.SpotOrderList, error) {
	direction := et.ListASC // 撮合按时间先后顺序
	price := marketDepth.Price
	if price != m.lastOrderPrice {
		m.orderKey = ""
		m.endOrderList = false
	}

	orderList, err := findOrderIDListByPrice(m.localdb, payload.GetLeftAsset(), payload.GetRightAsset(), price, OpSwap(payload.Op), direction, m.orderKey)
	if err != nil {
		if err == types.ErrNotFound {
			return &et.SpotOrderList{List: []*et.SpotOrder{}, PrimaryKey: ""}, nil
		}
		elog.Error("findOrderIDListByPrice error" /*"height", a.height, */, "symbol", payload.GetLeftAsset(), "price", marketDepth.Price, "op", OpSwap(payload.Op), "error", err)
		return nil, err
	}
	// reatch the last order list for price
	if orderList.PrimaryKey == "" {
		m.endOrderList = true
	}

	// set next key
	m.orderKey = orderList.PrimaryKey
	return orderList, nil
}

func (m *matcher) isEndOrderList(price int64) bool {
	return price == m.lastOrderPrice && m.endOrderList
}

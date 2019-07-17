// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
	auty "github.com/33cn/plugin/plugin/dapp/autonomy/types"

)



func (a *action) propRule(prob *auty.ProposalRule) (*types.Receipt, error) {
	//如果全小于等于0,则说明该提案规则参数不正确
	if prob.RuleCfg == nil || prob.RuleCfg.BoardAttendRatio <= 0 && prob.RuleCfg.BoardApproveRatio <= 0  &&
	   prob.RuleCfg.PubOpposeRatio <= 0 && prob.RuleCfg.ProposalAmount <= 0 && prob.RuleCfg.LargeProjectAmount <= 0 {
		return  nil, types.ErrInvalidParam
	}

	if prob.StartBlockHeight < a.height || prob.EndBlockHeight < a.height {
		return  nil, types.ErrInvalidParam
	}

	// 获取当前生效提案规则,并且将不修改的规则补齐
	rule, err := a.getActiveRule()
	if err != nil {
		alog.Error("propRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "getActiveRule failed", err)
		return nil, err
	}

	if prob.RuleCfg.BoardAttendRatio > 0 {
		rule.BoardAttendRatio = prob.RuleCfg.BoardAttendRatio
	}
	if prob.RuleCfg.BoardApproveRatio > 0  {
		rule.BoardApproveRatio = prob.RuleCfg.BoardApproveRatio
	}
	if prob.RuleCfg.PubOpposeRatio > 0 {
		rule.BoardApproveRatio = prob.RuleCfg.PubOpposeRatio
	}
	if prob.RuleCfg.ProposalAmount > 0{
		rule.ProposalAmount = prob.RuleCfg.ProposalAmount
	}
	if prob.RuleCfg.LargeProjectAmount > 0 {
		rule.LargeProjectAmount = prob.RuleCfg.LargeProjectAmount
	}

	receipt, err := a.coinsAccount.ExecFrozen(a.fromaddr, a.execaddr, rule.ProposalAmount)
	if err != nil {
		alog.Error("propRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "ExecFrozen amount", rule.ProposalAmount)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	cur := &auty.AutonomyProposalRule{
		PropRule:prob,
		Rule: rule,
		VoteResult: &auty.VoteResult{},
		Status: auty.AutonomyStatusProposalRule,
		Address: a.fromaddr,
		Height: a.height,
		Index: a.index,
	}

	key := propRuleID(common.ToHex(a.txhash))
	value := types.Encode(cur)
	kv = append(kv, &types.KeyValue{Key: key, Value: value})

	receiptLog := getRuleReceiptLog(nil, cur, auty.TyLogPropRule)
	logs = append(logs, receiptLog)

	return &types.Receipt{Ty: types.ExecOk, KV: kv, Logs: logs}, nil
}

func (a *action) rvkPropRule(rvkProb *auty.RevokeProposalRule) (*types.Receipt, error) {
	cur, err := a.getProposalRule(rvkProb.ProposalID)
	if err != nil {
		alog.Error("rvkPropRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "getProposalRule failed",
			rvkProb.ProposalID, "err", err)
		return nil, err
	}
	pre := copyAutonomyProposalRule(cur)

	// 检查当前状态
	if cur.Status != auty.AutonomyStatusProposalRule {
		err := auty.ErrProposalStatus
		alog.Error("rvkPropRule ", "addr", a.fromaddr, "status", cur.Status, "status is not match",
			rvkProb.ProposalID, "err", err)
		return nil, err
	}

	start := cur.GetPropRule().StartBlockHeight
	if a.height > start {
		err := auty.ErrRevokeProposalPeriod
		alog.Error("rvkPropRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "ProposalID",
			rvkProb.ProposalID, "err", err)
		return nil, err
	}

	if a.fromaddr != cur.Address {
		err := auty.ErrRevokeProposalPower
		alog.Error("rvkPropRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "ProposalID",
			rvkProb.ProposalID, "err", err)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	receipt, err := a.coinsAccount.ExecActive(a.fromaddr, a.execaddr, cur.Rule.ProposalAmount)
	if err != nil {
		alog.Error("rvkPropRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "ExecActive amount", cur.Rule.ProposalAmount, "err", err)
		return nil, err
	}
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	cur.Status = auty.AutonomyStatusRvkPropRule

	kv = append(kv, &types.KeyValue{Key: propRuleID(rvkProb.ProposalID), Value: types.Encode(cur)})

	getRuleReceiptLog(pre, cur, auty.TyLogRvkPropRule)

	return &types.Receipt{Ty: types.ExecOk, KV: kv, Logs: logs}, nil
}

func (a *action) votePropRule(voteProb *auty.VoteProposalRule) (*types.Receipt, error) {
	cur, err := a.getProposalRule(voteProb.ProposalID)
	if err != nil {
		alog.Error("votePropRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "getProposalRule failed",
			voteProb.ProposalID, "err", err)
		return nil, err
	}
	pre := copyAutonomyProposalRule(cur)

	// 检查当前状态
	if cur.Status != auty.AutonomyStatusProposalRule && cur.Status != auty.AutonomyStatusVotePropRule {
		err := auty.ErrProposalStatus
		alog.Error("votePropRule ", "addr", a.fromaddr, "status", cur.Status, "ProposalID",
			voteProb.ProposalID, "err", err)
		return nil, err
	}

	start := cur.GetPropRule().StartBlockHeight
	end := cur.GetPropRule().EndBlockHeight
	real := cur.GetPropRule().RealEndBlockHeight
	if start < a.height || end < a.height || (real != 0 && real < a.height) {
		err := auty.ErrVotePeriod
		alog.Error("votePropRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "ProposalID",
			voteProb.ProposalID, "err", err)
		return nil, err
	}

	// 检查是否已经参与投票
	votes, err := a.checkVotesRecord(voteProb.ProposalID)
	if err != nil {
		alog.Error("votePropRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "checkVotesRecord failed",
			voteProb.ProposalID, "err", err)
		return nil, err
	}
	// 更新投票记录
	votes.Address = append(votes.Address, a.fromaddr)

	if cur.GetVoteResult().TotalVotes == 0 { //需要统计票数
	    addr := "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
		account, err := a.getStartHeightVoteAccount(addr, start)
		if err != nil {
			return nil, err
		}
		cur.VoteResult.TotalVotes = int32(account.Balance/ticketPrice)
	}

	// 获取可投票数
	account, err := a.getStartHeightVoteAccount(a.fromaddr, start)
	if err != nil {
		return nil, err
	}
	if voteProb.Approve {
		cur.VoteResult.ApproveVotes +=  int32(account.Balance/ticketPrice)
	} else {
		cur.VoteResult.OpposeVotes += int32(account.Balance/ticketPrice)
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	if cur.VoteResult.TotalVotes != 0 &&
		cur.VoteResult.ApproveVotes + cur.VoteResult.OpposeVotes != 0 &&
	    float32(cur.VoteResult.ApproveVotes + cur.VoteResult.OpposeVotes) / float32(cur.VoteResult.TotalVotes) >= float32(pubAttendRatio)/100.0 &&
		float32(cur.VoteResult.ApproveVotes) / float32(cur.VoteResult.ApproveVotes + cur.VoteResult.OpposeVotes) >= float32(pubApproveRatio)/100.0 {
		cur.VoteResult.Pass = true
		cur.PropRule.RealEndBlockHeight = a.height

		receipt, err := a.coinsAccount.ExecTransferFrozen(cur.Address, autonomyAddr, a.execaddr, cur.Rule.ProposalAmount)
		if err != nil {
			alog.Error("votePropRule ", "addr", cur.Address, "execaddr", a.execaddr, "ExecTransferFrozen amount fail", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}

	key := propRuleID(voteProb.ProposalID)
	cur.Status = auty.AutonomyStatusVotePropRule
	if cur.VoteResult.Pass {
		cur.Status = auty.AutonomyStatusTmintPropRule
	}
	kv = append(kv, &types.KeyValue{Key: key, Value: types.Encode(cur)})

	// 更新VotesRecord
	kv = append(kv, &types.KeyValue{Key: VotesRecord(voteProb.ProposalID), Value: types.Encode(votes)})

	// 更新系统规则
	if cur.VoteResult.Pass {
		kv = append(kv, &types.KeyValue{Key: activeRuleID(), Value:types.Encode(cur.Rule)})
	}

	ty := auty.TyLogVotePropRule
	if cur.VoteResult.Pass {
		ty = auty.TyLogTmintPropRule
	}
	receiptLog := getRuleReceiptLog(pre, cur, int32(ty))
	logs = append(logs, receiptLog)

	return &types.Receipt{Ty: types.ExecOk, KV: kv, Logs: logs}, nil
}

func (a *action) tmintPropRule(tmintProb *auty.TerminateProposalRule) (*types.Receipt, error) {
	cur, err := a.getProposalRule(tmintProb.ProposalID)
	if err != nil {
		alog.Error("tmintPropRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "getProposalRule failed",
			tmintProb.ProposalID, "err", err)
		return nil, err
	}

	pre := copyAutonomyProposalRule(cur)

	// 检查当前状态
	if cur.Status == auty.AutonomyStatusTmintPropRule {
		err := auty.ErrProposalStatus
		alog.Error("tmintPropRule ", "addr", a.fromaddr, "status", cur.Status, "status is not match",
			tmintProb.ProposalID, "err", err)
		return nil, err
	}

	start := cur.GetPropRule().StartBlockHeight
	end := cur.GetPropRule().EndBlockHeight
	if a.height < end && !cur.VoteResult.Pass {
		err := auty.ErrTerminatePeriod
		alog.Error("tmintPropRule ", "addr", a.fromaddr, "status", cur.Status, "height", a.height,
			"in vote period can not terminate", tmintProb.ProposalID, "err", err)
		return nil, err
	}

	if cur.GetVoteResult().TotalVotes == 0 { //需要统计票数
		addr := "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
		account, err := a.getStartHeightVoteAccount(addr, start)
		if err != nil {
			return nil, err
		}
		cur.VoteResult.TotalVotes = int32(account.Balance/ticketPrice)
	}

	if float32(cur.VoteResult.ApproveVotes + cur.VoteResult.OpposeVotes) / float32(cur.VoteResult.TotalVotes) >=  float32(pubAttendRatio)/100.0 &&
		float32(cur.VoteResult.ApproveVotes) / float32(cur.VoteResult.ApproveVotes + cur.VoteResult.OpposeVotes) >= float32(pubApproveRatio)/100.0 {
		cur.VoteResult.Pass = true
	} else {
		cur.VoteResult.Pass = false
	}
	cur.PropRule.RealEndBlockHeight = a.height

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	receipt, err := a.coinsAccount.ExecTransferFrozen(cur.Address, autonomyAddr, a.execaddr, cur.Rule.ProposalAmount)
	if err != nil {
		alog.Error("votePropRule ", "addr", a.fromaddr, "execaddr", a.execaddr, "ExecTransferFrozen amount fail", err)
		return nil, err
	}
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	cur.Status = auty.AutonomyStatusTmintPropRule

	kv = append(kv, &types.KeyValue{Key: propRuleID(tmintProb.ProposalID), Value: types.Encode(cur)})

	// 更新系统规则
	if cur.VoteResult.Pass {
		kv = append(kv, &types.KeyValue{Key: activeRuleID(), Value:types.Encode(cur.Rule)})
	}

	getRuleReceiptLog(pre, cur, auty.TyLogTmintPropRule)

	return &types.Receipt{Ty: types.ExecOk, KV: kv, Logs: logs}, nil
}

func (a *action) getProposalRule(ID string) (*auty.AutonomyProposalRule, error) {
	value, err := a.db.Get(propRuleID(ID))
	if err != nil {
		return nil, err
	}
	cur := &auty.AutonomyProposalRule{}
	err = types.Decode(value, cur)
	if err != nil {
		return nil, err
	}
	return cur, nil
}

// getReceiptLog 根据提案信息获取log
// 状态变化：
func getRuleReceiptLog(pre, cur *auty.AutonomyProposalRule, ty int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = ty
	r := &auty.ReceiptProposalRule{Prev: pre, Current: cur}
	log.Log = types.Encode(r)
	return log
}

func copyAutonomyProposalRule(cur *auty.AutonomyProposalRule) *auty.AutonomyProposalRule {
	newAut := *cur
	newRule := *cur.GetPropRule()
	newCfg := *cur.GetPropRule().GetRuleCfg()
	newRes := *cur.GetVoteResult()
	newAut.PropRule = &newRule
	newAut.PropRule.RuleCfg = &newCfg
	newAut.VoteResult = &newRes
	return &newAut
}

package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/ebay/libovsdb"
)

// ACLRuleAttr ...
const (
	ACLRuleAttrMatchSRCMAC       = "aclrule_match_srcmac"
	ACLRuleAttrMatchDSTMAC       = "aclrule_match_dstmac"
	ACLRuleAttrMatchETHERTYPE    = "aclrule_match_ethertype"
	ACLRuleAttrMatchSRCIP        = "aclrule_match_srcip"
	ACLRuleAttrMatchSRCMASK      = "aclrule_match_srcmask"
	ACLRuleAttrMatchDSTIP        = "aclrule_match_dstip"
	ACLRuleAttrMatchDSTMASK      = "aclrule_match_dstmask"
	ACLRuleAttrMatchPROTOCOL     = "aclrule_match_protocol"
	ACLRuleAttrMatchSRCPORTMIN   = "aclrule_match_srcportmin"
	ACLRuleAttrMatchSRCPORTMAX   = "aclrule_match_srcportmax"
	ACLRuleAttrMatchDSTPORTMIN   = "aclrule_match_dstportmin"
	ACLRuleAttrMatchDSTPORTMAX   = "aclrule_match_dstportmax"
	ACLRuleAttrMatchTCPFLAGS     = "aclrule_match_tcpflags"
	ACLRuleAttrMatchTCPFLAGSMASK = "aclrule_match_tcpflagmask"
	ACLRuleAttrMatchICMPTYPE     = "aclrule_match_icmptype"
	ACLRuleAttrMatchICMPCODE     = "aclrule_match_icmpcode"
	ACLRuleAttrAction            = "aclrule_action"
)

// ACLRuleObj ...
type ACLRuleObj struct {
	ACLName  string
	Sequence int
}

func rowToACLRuleObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableACLRule := vtepdb.ConvertRowToACLRule(libovsdb.ResultRow(row.Fields))

	obj := ACLRuleObj{
		ACLName:  tableACLRule.ACLName,
		Sequence: tableACLRule.Sequence,
	}
	attrs := map[interface{}]interface{}{
		ACLRuleAttrMatchSRCMAC:       tableACLRule.SourceMac,
		ACLRuleAttrMatchDSTMAC:       tableACLRule.DestMac,
		ACLRuleAttrMatchETHERTYPE:    tableACLRule.Ethertype,
		ACLRuleAttrMatchSRCIP:        tableACLRule.SourceIP,
		ACLRuleAttrMatchSRCMASK:      tableACLRule.SourceMask,
		ACLRuleAttrMatchDSTIP:        tableACLRule.DestIP,
		ACLRuleAttrMatchDSTMASK:      tableACLRule.DestMask,
		ACLRuleAttrMatchPROTOCOL:     tableACLRule.Protocol,
		ACLRuleAttrMatchSRCPORTMIN:   tableACLRule.SourcePortMin,
		ACLRuleAttrMatchSRCPORTMAX:   tableACLRule.SourcePortMax,
		ACLRuleAttrMatchDSTPORTMIN:   tableACLRule.DestPortMin,
		ACLRuleAttrMatchDSTPORTMAX:   tableACLRule.DestPortMax,
		ACLRuleAttrMatchTCPFLAGS:     tableACLRule.TCPFlags,
		ACLRuleAttrMatchTCPFLAGSMASK: tableACLRule.TCPFlagsMask,
		ACLRuleAttrMatchICMPTYPE:     tableACLRule.IcmpType,
		ACLRuleAttrMatchICMPCODE:     tableACLRule.IcmpCode,
		ACLRuleAttrAction:            tableACLRule.Action,
	}
	return obj, attrs
}

func rowToACLRuleAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.ACLRuleFieldSourceMac:
			if srcMac, ok := value.(string); ok {
				attrs[ACLRuleAttrMatchSRCMAC] = srcMac
			}
		case vtepdb.ACLRuleFieldDestMac:
			if dstMac, ok := value.(string); ok {
				attrs[ACLRuleAttrMatchDSTMAC] = dstMac
			}
		}
	}

	return attrs
}

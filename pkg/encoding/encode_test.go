package encoding

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"testing"
)

type cmdMatchStart struct {
	RoleID             uint32
	Info               string
	RoleType           int32 // 0:Police 1:Thief
	AuthKey            string
	AreaID             int32 // 客户端建议的 areaID
	Score              int32
	TeamID             int32
	TeamMemberNum      int32
	CharacterInfo      string
	PlayerData         string
	PlayerDataAuthKey  string
	PerformanceQuality byte
	DIYParam           string
}

func getBenchmarkMarshalData() *cmdMatchStart {
	return &cmdMatchStart{
		RoleID:             10055310,
		Info:               "{\"name\":\"崩撤卖溜\",\"ip\":\"119.121.154.71\",\"area\":\"中国 广东\",\"gradeThief\":111,\"gradePointThief\":121,\"gradePolice\":41,\"gradePointPolice\":54,\"vipInfo\":{\"pfID\":0,\"isVip\":false,\"isSuperVip\":false,\"isYearVip\":false,\"vipLevel\":0},\"icon\":\"0\",\"sex\":0,\"age\":16,\"province\":6,\"gm\":false,\"membershipInfo\":[{\"membershipID\":1,\"expireTime\":0},{\"membershipID\":2,\"expireTime\":0}],\"activeHeadBoxID\":900001,\"activeBubbleBoxID\":910001,\"unionID\":353,\"unionName\":\"锁功能Test\",\"unionBadge\":{\"iconID\":101,\"frameID\":1001,\"flagID\":10001}}",
		RoleType:           1,
		AuthKey:            "7f33a39afb7a59505d0fa38e35f30e76",
		AreaID:             1,
		Score:              10100690,
		TeamID:             0,
		TeamMemberNum:      0,
		CharacterInfo:      "{\"characterID\":200,\"expiredTime\":0,\"currentSkinInfo\":{\"skinPartIDs\":[2001,2002,2003,2004,2005],\"skinPartColors\":[12,38,25,51,54],\"expiredTime\":[0,0,0,0,0]},\"ExpLevel\":2,\"ExpPoint\":25,\"TalentPointRemained\":1,\"TalentLevels\":[1,1,1]}",
		PlayerData:         "{\"cards\":[104,103,100,200,107],\"cardLevel\":[1,1,2,1,1],\"cardSkins\":[500005,500004,500001,500010,500008],\"cardStyles\":[0,0,0,0,0],\"ingameEmotion\":[950001,960011,0,0,0,0],\"lightnessConfig\":0,\"lockCardConfig\":[2,1,0],\"ai\":0,\"blocked\":false}",
		PlayerDataAuthKey:  "911eec312b8102a887ef423cd4dd1e76",
		PerformanceQuality: 100,
		DIYParam:           "{\"fail_to_boss\":0,\"ts\":1601559565}",
	}
}

func BenchmarkMarshal(b *testing.B) {
	v := getBenchmarkMarshalData()
	for i := 0; i < b.N; i++ {
		data, err := Marshal(v)
		if err != nil {
			panic(err)
		}
		if len(data) == 0 {
			panic("empty result")
		}
	}
}

func BenchmarkJsonMarshal(b *testing.B) {
	v := getBenchmarkMarshalData()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		if len(data) == 0 {
			panic("empty result")
		}
	}
}

func BenchmarkGobMarshal(b *testing.B) {
	v := getBenchmarkMarshalData()
	data := &bytes.Buffer{}
	for i := 0; i < b.N; i++ {
		data.Reset()
		e := gob.NewEncoder(data)
		err := e.Encode(v)
		if err != nil {
			panic(err)
		}
		if data.Len() == 0 {
			panic("empty result")
		}
	}
}

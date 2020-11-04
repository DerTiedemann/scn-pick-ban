package lcu

// client Representation of a Summoner
type Summoner struct {
	DisplayName string
	Id          int64
}

// client Representation of a Cell in the PickBan Screen
type Cell struct {
	Id int64

	ChampionId int64
	SummonerId int64

	spell1Id int64
	spell2Id int64
}

type ActionType string

const (
	ActionPick ActionType = "PICK"
	ActionBan  ActionType = "BAN"
)

type Action struct {
	Completed  bool
	ChampionId int64
	Type       ActionType
	CellId     int64
}

type PickBanProvider interface {
	BlueTeam() []Summoner
	RedTeam() []Summoner

	Actions() <-chan Action

	IsInChampSelect() bool
}

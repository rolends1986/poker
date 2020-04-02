package table

// An Action is an action a player can take in a hand.
type Action string

const (
	// Fold discards one's hand and forfeits interest in
	// the current pot.
	// 收牌 / 不跟 - 放弃继续牌局的机会
	Fold Action = "Fold"

	// Check is the forfeit to bet when not faced with a bet or
	// raise.
	// 让牌 - 在无需跟进的情况下选择把决定“让”给下一位
	Check Action = "Check"

	// Call is a match of a bet or raise.
	// 跟进 - 跟随众人押上同等的注额
	Call Action = "Call"

	// Bet is a wager that others must match to remain a contender
	// in the current pot.
	// 押注 - 押上筹码
	Bet Action = "Bet"

	// Raise is an increase to the original bet that others must
	// match to remain a contender in the current pot.
	// 加注 - 把现有的注金抬高
	Raise Action = "Raise"

	// 牌局中玩家马上站起
	Stand Action = "stand"
)

// Player represents a player at a table.
type Player interface {
	// ID returns the unique identifier of the player.
	ID() int64
	Nickname() string
	Country() string
	// 是否站起
	Stand() bool
	// 是否托管
	Hosted() bool
	// 已游戏时长
	PlayDuration() int64

	// FromID resets the player from an id.  It is required for
	// deserialization.
	FromID(id int64) (Player, error)

	// Action returns the action and it's chip amount.  This method
	// will block table's Next() function until input is recieved.
	Action() (a Action, chips int, timeout bool, ignore bool)

	SaveAction(round int, playerAction PlayerAction)
}

// RegisterPlayer stores the player implementation for json deserialization.
func RegisterPlayer(p Player) {
	registeredPlayer = p
}

var (
	// mapping to player implemenation
	registeredPlayer Player
)

// Stakes are the forced bet amounts for the table.
type Stakes struct {

	// SmallBet is the smaller forced bet amount.
	SmallBet int `json:"smallBet" bson:"smallBet"`

	// BigBet is the bigger forced bet amount.
	BigBet int `json:"bigBet" bson:"bigBet"`

	// Ante is the amount requried from each player to start the hand.
	Ante int `json:"ante" bson:"ante"`

	// 强制Straddle标记
	Straddle bool `json:"straddle" bson:"straddle"`
}

// Limit is the bet and raise limits of a poker game
type Limit string

const (
	// NoLimit has no limit and players may go "all in"
	NoLimit Limit = "NL"

	// PotLimit has the current value of the pot as the limit
	PotLimit Limit = "PL"

	// FixedLimit restricted the size of bets and raises to predefined
	// values based on the game and round.
	FixedLimit Limit = "FL"
)

// Config are the configurations for creating a table.
type Config struct {

	// Game is the game of the table.
	Game Game `json:"game" bson:"game"`

	// Limit is the limit of the table
	Limit Limit `json:"limit" bson:"limit"`

	// Stakes is the stakes for the table.
	Stakes Stakes `json:"stakes" bson:"stakes"`

	// NumOfSeats is the number of seats available for the table.
	NumOfSeats int `json:"numOfSeats" bson:"numOfSeats"`
}

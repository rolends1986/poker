package table

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/rolends1986/poker/hand"
)

var (
	// ErrInvalidBuyin errors occur when a player attempts to sit at a
	// table with an invalid buyin.
	ErrInvalidBuyin = errors.New("table: player attempted sitting with invalid buyin")

	// ErrSeatOccupied errors occur when a player attempts to sit at a
	// table in a seat that is already occupied.
	ErrSeatOccupied = errors.New("table: player attempted sitting in occupied seat")

	// ErrInvalidSeat errors occur when a player attempts to sit at a
	// table in a seat that is invalid.
	ErrInvalidSeat = errors.New("table: player attempted sitting in invalid seat")

	// ErrAlreadySeated errors occur when a player attempts to sit at a
	// table at which the player is already seated.
	ErrAlreadySeated = errors.New("table: player attempted sitting when already seated")

	// ErrInsufficientPlayers errors occur when the table's Next() method
	// can't start a new hand because of insufficient players
	ErrInsufficientPlayers = errors.New("table: insufficent players for call to table's Next() method")

	// ErrInvalidBet errors occur when a player attempts to bet an invalid
	// amount.  Bets are invalid if they exceed a player's chips or fall below the
	// stakes minimum bet.  In fixed limit games the bet amount must equal the amount
	// prespecified by the limit and round.  In pot limit games the bet must be less
	// than or equal to the pot.
	ErrInvalidBet = errors.New("table: player attempted invalid bet")

	// ErrInvalidRaise errors occur when a player attempts to raise an invalid
	// amount.  Raises are invalid if the raise or reraise is lower than the previous bet
	// or raised amount unless it puts the player allin.  Raises are also invalid if they
	// exceed a player's chips. In fixed limit games the raise amount must equal the amount
	// prespecified by the limit and round.  In pot limit games the raise must be less
	// than or equal to the pot.
	ErrInvalidRaise = errors.New("table: player attempted invalid raise")

	// ErrInvalidAction errors occur when a player attempts an action that isn't
	// currently allowed.  For example a check action is invalid when faced with a raise.
	ErrInvalidAction = errors.New("table: player attempted invalid action")
)

type StraddleCategory uint8

const (
	Straddle1 StraddleCategory = 1
	Straddle2 StraddleCategory = 2
	Straddle3 StraddleCategory = 3
)

type StraddleSeat struct {
	UserId    int64            `json:"-"`
	Seat      int              `json:"seat"`
	Category  StraddleCategory `json:"category"`
	Voluntary bool             `json:"voluntary"`
}

type PlayerAction struct {
	PlayerId   int64     `json:"playerId" bson:"playerId"`
	Action     Action    `json:"action" bson:"action"`
	Chips      int       `json:"chips" bson:"chips"`
	ActionTime time.Time `json:"actionTime" bson:"actionTime"`
	Timeout    bool      `json:"timeout" bson:"timeout"`
	RoundPot   int       `json:"roundPot" bson:"roundPot"`
	Pot        int       `json:"pot" bson:"pot"`
}

// PlayerState is the state of a player at a table.
type PlayerState struct {
	player     Player
	holeCards  []*HoleCard
	chips      int
	beginChips int // 开始筹码，玩家在每一手开始时的筹码。
	acted      bool
	out        bool
	allin      bool
	canRaise   bool
	roundPot   int  // 对应 round 的下注总额
	pot        int  // 当前手的下注总额
	stand      bool // 玩家是否站起
	straddle   bool // 是否下一手自愿straddle
}

// Acted returns whether or not the player has acted for the current round.
func (state *PlayerState) Acted() bool {
	return state.acted
}

// AllIn returns whether or not the player is all in for the current hand.
func (state *PlayerState) AllIn() bool {
	return state.allin
}

// CanRaise returns whether or not the player can raise in the current round.
func (state *PlayerState) CanRaise() bool {
	return state.canRaise
}

// Chips returns the number of chips the player has in his or her stack.
func (state *PlayerState) Chips() int {
	return state.chips
}

func (state *PlayerState) BeginChips() int {
	return state.beginChips
}

// HoleCards returns the hole cards the player currently has.
func (state *PlayerState) HoleCards() []*HoleCard {
	c := []*HoleCard{}
	return append(c, state.holeCards...)
}

// Out returns whether or not the player is out of the current hand.
func (state *PlayerState) Out() bool {
	return state.out
}

// Player returns the player.
func (state *PlayerState) Player() Player {
	return state.player
}

func (state *PlayerState) Pot() int {
	return state.pot
}

func (state *PlayerState) SetStraddle(value bool) {
	state.straddle = value
}

func (state *PlayerState) GetStraddle() bool {
	return state.straddle
}

// deduct 表示翻牌前玩家roundPot中忽略前注
func (state *PlayerState) addToPot(chips, deduct int, r int) {
	if round(r) == preflop {
		state.roundPot += chips - deduct
	} else {
		state.roundPot += chips
	}
	state.pot += chips
}

func (state *PlayerState) MarkStand() {
	state.stand = true
}

// String returns a string useful for debugging.
func (state *PlayerState) String() string {
	const format = "{Player: %v, Chips: %d, Acted: %t, Out: %t, AllIn: %t, RoundPot: %d, Pot: %d, beginChips: %d}"
	return fmt.Sprintf(format,
		state.player.ID(), state.chips, state.acted, state.out, state.allin, state.roundPot, state.pot, state.beginChips)
}

func (state *PlayerState) PlayerStateJSON() PlayerStateJSON {
	player := state.Player()
	return PlayerStateJSON{
		ID:           player.ID(),
		Nickname:     player.Nickname(),
		Country:      player.Country(),
		Hosted:       player.Hosted(),
		PlayDuration: player.PlayDuration(),
		HoleCards:    state.HoleCards(),
		Chips:        state.Chips(),
		BeginChips:   state.BeginChips(),
		Acted:        state.Acted(),
		Out:          state.Out(),
		Allin:        state.AllIn(),
		RoundPot:     state.roundPot,
		Pot:          state.pot,
		CanRaise:     state.CanRaise(),
		Stand:        state.stand,
	}
}

type PlayerStateJSON struct {
	ID           int64       `json:"id" bson:"id"`
	Nickname     string      `json:"nickname" bson:"nickname"`
	Country      string      `json:"country" bson:"country"` // 对应的国家代码 ISO 3166
	Hosted       bool        `json:"hosted" bson:"hosted"`
	RoundPot     int         `json:"roundPot" bson:"roundPot"`
	Pot          int         `json:"pot" bson:"pot"`
	HoleCards    []*HoleCard `json:"holeCards" bson:"holeCards"`
	Chips        int         `json:"chips" bson:"chips"`
	BeginChips   int         `json:"beginChips" bson:"beginChips"`
	Acted        bool        `json:"acted" bson:"acted"`
	Out          bool        `json:"out" bson:"out"`
	Allin        bool        `json:"allin" bson:"allin"`
	CanRaise     bool        `json:"canRaise" bson:"canRaise"`
	Stand        bool        `json:"stand" bson:"stand"`
	Straddle     bool        `json:"straddle" bson:"straddle"`
	PlayDuration int64       `json:"playDuration" bson:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (state *PlayerState) MarshalJSON() ([]byte, error) {
	// player := state.Player()
	// tpJSON := &PlayerStateJSON{
	// 	ID:        player.ID(),
	// 	Nickname:  player.Nickname(),
	// 	Avatar:    player.Avatar(),
	// 	HoleCards: state.HoleCards(),
	// 	Chips:     state.Chips(),
	// 	Acted:     state.Acted(),
	// 	Out:       state.Out(),
	// 	Allin:     state.AllIn(),
	// 	RoundPot:  state.roundPot,
	// 	CanRaise:  state.CanRaise(),
	// }
	tpJSON := state.PlayerStateJSON()
	return json.Marshal(tpJSON)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (state *PlayerState) UnmarshalJSON(b []byte) error {
	tpJSON := &PlayerStateJSON{}
	if err := json.Unmarshal(b, tpJSON); err != nil {
		return err
	}

	if isNil(registeredPlayer) {
		return errors.New("table: PlayerState json deserialization requires use of the RegisterPlayer function")
	}

	p, err := registeredPlayer.FromID(tpJSON.ID)
	if err != nil {
		return fmt.Errorf("table PlayerState json deserialization failed because of player %s FromID - %s", tpJSON.ID, err)
	}

	state.player = p
	state.holeCards = tpJSON.HoleCards
	state.chips = tpJSON.Chips
	state.beginChips = tpJSON.BeginChips
	state.acted = tpJSON.Acted
	state.out = tpJSON.Out
	state.allin = tpJSON.Allin
	state.canRaise = tpJSON.CanRaise
	state.roundPot = tpJSON.RoundPot
	state.pot = tpJSON.Pot
	state.stand = tpJSON.Stand
	state.straddle = tpJSON.Straddle

	return nil
}

// Table represent a poker table and dealer.  A table manages the
// game state and all player interactions at the table.
type Table struct {
	opts          Config
	dealer        hand.Dealer
	deck          *hand.Deck
	button        int
	smallBetSeat  int // 小盲位
	bigBetSeat    int // 大盲位
	utgSeat       int // 枪口位
	action        int
	round         int
	minRaise      int // 玩家加注值=下注值-roundPot-outstanding
	board         []*hand.Card
	players       map[int]*PlayerState
	pot           *Pot
	sidePots      []*Pot
	startedHand   bool
	showdown      bool            // 是否可以摊牌
	straddleSeats []*StraddleSeat // 本轮straddle位
	sync.RWMutex  `bson:"-" json:"-"`
}

// New creates a new table with the options and deck provided.  To
// start playing hands, at least two players must be seated and the
// Next() function must be called.  If the number of seats is invalid
// for the Game specified New panics.
func New(opts Config, dealer hand.Dealer) *Table {
	if int(opts.NumOfSeats) > opts.Game.get().MaxSeats() {
		format := "table: %s has a maximum of %d seats but attempted %d"
		s := fmt.Sprintf(format, opts.Game, opts.Game.get().MaxSeats(), opts.NumOfSeats)
		panic(s)
	}

	return &Table{
		opts:          opts,
		dealer:        dealer,
		deck:          dealer.Deck(),
		board:         []*hand.Card{},
		players:       map[int]*PlayerState{},
		pot:           newPot(int(opts.NumOfSeats)),
		action:        -1,
		straddleSeats: []*StraddleSeat{},
	}
}

// 获取玩家的 beginChips
func (t *Table) GetPlayerBeginChips() map[int]int {
	playerBeginChips := map[int]int{}

	for seat, player := range t.Players() {
		playerBeginChips[seat] = player.BeginChips()
	}

	return playerBeginChips
}

// 涨盲
func (t *Table) RiseBlinds(smallBet, bigBet int) {
	t.opts.Stakes.SmallBet = smallBet
	t.opts.Stakes.BigBet = bigBet
}

func (t *Table) SmallBet() int {
	return t.opts.Stakes.SmallBet
}

func (t *Table) BigBet() int {
	return t.opts.Stakes.BigBet
}

func (t *Table) Ante() int {
	return t.opts.Stakes.Ante
}

// Action returns the seat that the action is currently on.  If no
// seat has the action then -1 is returned.
func (t *Table) Action() int {
	return t.action
}

// Board returns the current community cards.  An empty slice is
// returned if there are no community cards or the game doesn't
// support community cards.
func (t *Table) Board() []*hand.Card {
	c := []*hand.Card{}
	return append(c, t.board...)
}

func (t *Table) BoardString() []string {
	c := []string{}
	for _, item := range t.board {
		c = append(c, item.String())
	}
	return c
}

// Button returns the seat that the button is currently on.
func (t *Table) Button() int {
	return t.button
}

// returns the seat that the small bet is currently on.
func (t *Table) SmallBetSeat() int {
	return t.smallBetSeat
}

// returns the seat that the big bet is currently on.
func (t *Table) BigBetSeat() int {
	return t.bigBetSeat
}

// returns the seat that the big bet is currently on.
func (t *Table) UtgSeat() int {
	return t.utgSeat
}

// CurrentPlayer returns the player the action is currently on.  If
// no player is current then it returns nil.
func (t *Table) CurrentPlayer() *PlayerState {
	t.RLock()
	defer t.RUnlock()
	return t.players[t.Action()]
}

// Game returns the game of the table.
func (t *Table) Game() Game {
	return t.opts.Game
}

// Limit returns the limit of the table.
func (t *Table) Limit() Limit {
	return t.opts.Limit
}

// MaxRaise returns the maximum number of chips that can be bet or
// raised by the current player.  If there is no current player then
// -1 is returned.
func (t *Table) MaxRaise() int {
	player := t.CurrentPlayer()
	if isNil(player) {
		return -1
	}

	outstanding := t.Outstanding()
	chips := player.Chips()
	//bettableChips := chips - outstanding
	// 玩家此轮总筹码
	bettableChips := chips + player.roundPot

	if bettableChips <= 0 {
		return 0
	}

	max := bettableChips
	switch t.opts.Limit {
	case PotLimit:
		// 1倍底池 = 平跟后的总底池 + outstanding + roundPot
		// 平跟后的总底池 = pot.chips + outstanding
		max = t.pot.Chips() + outstanding + outstanding + player.roundPot
	case FixedLimit:
		max = t.game().FixedLimit(t.opts, round(t.round))
	}
	// 最大下注不能大于总筹码
	if max > bettableChips {
		max = bettableChips
	}
	return max
}

// MinRaise returns the minimum number of chips that can be bet or
// raised by the current player. If there is no current player then
// -1 is returned.
func (t *Table) MinRaise() int {
	player := t.CurrentPlayer()
	if isNil(player) {
		return -1
	}

	outstanding := t.Outstanding()
	// 玩家此轮总筹码
	bettableChips := player.Chips() + player.roundPot

	switch t.opts.Limit {
	case FixedLimit:
		return 1
	}

	// 当底池中只有盲注或跟注盲注的动作时，这时加注最小值为盲注的两倍
	// 再加注的最小加注=你跟注他所需筹码量(outstanding)+他的加注部分(t.minRaise)+你的roundPot
	// 最后一个Straddle并不视为盲注，而是视为raise
	min := 0
	if t.IsPreFlop() {
		if t.minRaise == 0 {
			// 底池中只有盲注或跟注盲注的动作
			min = 2 * t.opts.Stakes.BigBet
		} else {
			// 翻牌前已有玩家加注
			min = outstanding + t.minRaise + player.roundPot
		}
	} else {
		if t.minRaise == 0 {
			// 翻牌/河牌/转牌圈第一个操作玩家
			min = t.opts.Stakes.BigBet
		} else {
			// 翻牌/河牌/转牌圈已有玩家加注
			min = outstanding + t.minRaise + player.roundPot
		}
	}

	if bettableChips < min {
		min = bettableChips
	}
	return min
}

// NumOfSeats returns the number of seats.
func (t *Table) NumOfSeats() int {
	return int(t.opts.NumOfSeats)
}

// Outstanding returns the number of chips owed to the pot by the
// current player.  If there is no current player then -1 is returned.
func (t *Table) Outstanding() int {
	player := t.CurrentPlayer()
	if isNil(player) {
		return -1
	}
	if player.AllIn() || player.Out() {
		return 0
	}
	return t.pot.Outstanding(t.Action())
}

// Players returns a mapping of seats to player states.  Empty seats
// are not included.
func (t *Table) Players() map[int]*PlayerState {
	t.RLock()
	defer t.RUnlock()
	players := map[int]*PlayerState{}
	for seat, p := range t.players {
		players[seat] = p
	}
	return players
}

func (t *Table) Player(seat int) *PlayerState {
	t.RLock()
	defer t.RUnlock()
	return t.players[seat]
}

func (t *Table) IsOut(seat int) bool {
	state := t.Player(seat)
	if state == nil {
		return true
	}
	return state.Out()
}

func (t *Table) IsPreFlop() bool {
	return t.round == 0
}

func (t *Table) IsFlop() bool {
	return t.round == 1
}

func (t *Table) IsTurn() bool {
	return t.round == 2
}

func (t *Table) IsRiver() bool {
	return t.round == 3
}

// 判断是否可以摊牌
func (t *Table) Showdown() bool {
	return t.showdown
}

// View returns a view of the table that only contains information
// privileged to the given player.
func (t *Table) View(p Player) *Table {
	t.RLock()
	defer t.RUnlock()
	players := map[int]*PlayerState{}
	for seat, player := range t.players {
		if p.ID() == player.Player().ID() {
			// 玩家查看自己手牌应设置Visibility=Exposed
			pCopy := new(PlayerState)
			*pCopy = *player
			pCopy.holeCards = make([]*HoleCard, 0, len(player.holeCards))
			for _, card := range player.holeCards {
				tmp := &HoleCard{Card: card.Card, Visibility: Exposed}
				pCopy.holeCards = append(pCopy.holeCards, tmp)
			}
			players[seat] = pCopy
			continue
		}

		if t.Showdown() && !player.out {
			players[seat] = player
			continue
		}

		players[seat] = &PlayerState{
			player:     player.player,
			holeCards:  tableViewOfHoleCards(player.holeCards),
			chips:      player.chips,
			beginChips: player.beginChips,
			acted:      player.acted,
			out:        player.out,
			allin:      player.allin,
			canRaise:   player.canRaise,
			roundPot:   player.roundPot,
			pot:        player.pot,
			stand:      player.stand,
		}
	}

	return &Table{
		opts:         t.opts,
		deck:         &hand.Deck{Cards: []*hand.Card{}},
		button:       t.button,
		action:       t.action,
		round:        t.round,
		minRaise:     t.minRaise,
		board:        t.board,
		pot:          t.pot,
		sidePots:     t.sidePots,
		startedHand:  t.startedHand,
		players:      players,
		smallBetSeat: t.smallBetSeat,
		bigBetSeat:   t.bigBetSeat,
		utgSeat:      t.utgSeat,
	}
}

// View returns a view of the table that for looker
func (t *Table) LookerView() *Table {
	players := map[int]*PlayerState{}
	t.RLock()
	for seat, player := range t.players {
		if t.Showdown() && !player.out {
			players[seat] = player
			continue
		}

		players[seat] = &PlayerState{
			player:     player.player,
			holeCards:  tableViewOfHoleCards(player.holeCards),
			chips:      player.chips,
			beginChips: player.beginChips,
			acted:      player.acted,
			out:        player.out,
			allin:      player.allin,
			canRaise:   player.canRaise,
			roundPot:   player.roundPot,
			pot:        player.pot,
			stand:      player.stand,
		}
	}
	t.RUnlock()

	return &Table{
		opts:         t.opts,
		deck:         &hand.Deck{Cards: []*hand.Card{}},
		button:       t.button,
		action:       t.action,
		round:        t.round,
		minRaise:     t.minRaise,
		board:        t.board,
		pot:          t.pot,
		sidePots:     t.sidePots,
		startedHand:  t.startedHand,
		players:      players,
		smallBetSeat: t.smallBetSeat,
		bigBetSeat:   t.bigBetSeat,
		utgSeat:      t.utgSeat,
	}
}

func (t *Table) DebugView() string {
	j, _ := json.Marshal(t.LookerView())
	return string(j)
}

func (t *Table) PrintTable() {
	players := t.Players()
	fmt.Println("")
	fmt.Println("-----Table-----")
	fmt.Println(t)
	for key, value := range players {
		fmt.Println(key, value)
	}
	//fmt.Printf("actions records: %v\n", actionsRecords[tbl.Round()])
	fmt.Printf("\nside pots: \n%v\n", t.sidePots)
	fmt.Println("-----Table-----")
	fmt.Println("")
}

func (t *Table) PrintResults(results map[int][]*Result) {
	players := t.Players()
	for seat, resultList := range results {
		for _, result := range resultList {
			fmt.Println(players[seat].Player().ID(), ":", result)
		}
	}
}

func (t *Table) Opts() Config {
	return t.opts
}

// Pot returns the current pot.
func (t *Table) Pot() *Pot {
	return t.pot
}

// returns the current side pots
func (t *Table) SidePots() []*Pot {
	return t.sidePots
}

// Round returns the current round.
func (t *Table) Round() int {
	return t.round
}

// Stakes returns the stakes.
func (t *Table) Stakes() Stakes {
	return t.opts.Stakes
}

// String returns a string useful for debugging.
func (t *Table) String() string {
	const format = "{Button: Seat %d, Current Player: %v, Round %d, Board: %s, Pot: %d}\n"
	var current int64 = 0
	if t.action != -1 && !isNil(t.CurrentPlayer()) {
		current = t.CurrentPlayer().player.ID()
	}

	return fmt.Sprintf(format, t.button, current, t.round, t.board, t.pot.Chips())
}

// ValidActions returns the actions that can be taken by the current
// player.
func (t *Table) ValidActions() []Action {
	player := t.CurrentPlayer()
	if player.AllIn() || player.Out() {
		return []Action{}
	}

	if t.Outstanding() == 0 {
		return []Action{Fold, Check, Bet}
	}

	if !player.CanRaise() {
		return []Action{Fold, Call}
	}

	if player.Chips() <= t.Outstanding() {
		return []Action{Fold, Call}
	}
	return []Action{Fold, Call, Raise}
}

func (t *Table) resetRoundPot() {
	t.Lock()
	defer t.Unlock()
	for _, p := range t.players {
		p.roundPot = 0
	}
}

func (t *Table) resetPot() {
	t.Lock()
	defer t.Unlock()
	for _, p := range t.players {
		p.roundPot = 0
		p.pot = 0
	}
}

// Next is the iterator function of the table.  Next updates the
// table's state while calling player's Action() method to get
// an action for each player's turn.  New hands are started
// automatically if there are two or more eligible players.  Next
// moves through each round of betting until the showdown at which
// point are paid out.  The results are returned as a map of seats
// to pot results. If the round is not a showdown then results are
// nil. err is nil unless there are insufficient players to start
// the next hand or a player's action has an error. done indicates
// that the table can not continue.
func (t *Table) Next() (results map[int][]*Result, done bool, err error) {
	if !t.startedHand {
		t.showdown = false
		t.resetPot()
		if !t.hasNextHand() {
			return nil, true, ErrInsufficientPlayers
		}
		t.setUpHand()
		t.setUpRound()
		t.startedHand = true
		return nil, false, nil
	}

	if t.action == -1 {
		t.round++
		t.resetRoundPot()

		if t.round == t.game().NumOfRounds() {
			holeCards := cardsFromHoleCardMap(t.HoleCards())
			highHands := newHands(holeCards, t.board, t.game().FormHighHand)
			lowHands := newHands(holeCards, t.board, t.game().FormLowHand)
			results = t.pot.payout(0, t, highHands, lowHands, t.game().Sorting(), t.button)
			t.payoutResults(results)
			t.startedHand = false
			t.action = -1
			t.showHoleCards()
			return results, false, nil
		}

		t.setUpRound()
		return nil, false, nil
	}

	current := t.CurrentPlayer()
	action, chips, timeout, ignore := current.player.Action()
	if !ignore {
		if err := t.handleAction(t.action, current, action, chips, timeout); err != nil {
			return nil, false, err
		}
	} else {
		log.WithFields(log.Fields{
			"userId":  current.player.ID(),
			"action":  action,
			"chips":   chips,
			"timeout": timeout,
		}).Info("Next: ignore player action")
	}

	// check if only one person left
	if t.EveryoneFolded() {
		for seat, player := range t.Players() {
			if player.out || player.stand {
				continue
			}
			results = t.pot.take(seat)
			t.payoutResults(results)
			t.startedHand = false
			t.action = -1
			return results, false, nil
		}

		// 另一个玩家站起时本玩家弃牌的并发场景，牌桌中
		// 可分配底池的玩家数为0
		view := t.LookerView()
		viewJson, _ := view.MarshalJSON()
		log.WithFields(log.Fields{
			"userId":  current.player.ID(),
			"action":  action,
			"chips":   chips,
			"timeout": timeout,
			"tbl":     string(viewJson),
		}).Warning("EveryoneFolded without payout")
		results = map[int][]*Result{}
		t.startedHand = false
		t.action = -1
		return results, false, nil
	}

	t.action = t.nextSeat(t.action+1, true)
	return nil, false, nil
}

// Sit sits the player at the table with the given amount of chips.
// An error is return if the seat is invalid, the player is already
// seated, the seat is already occupied, or the chips are outside
// the valid buy in amounts.
func (t *Table) Sit(p Player, seat, chips int, straddle bool) error {
	if !t.validSeat(seat) {
		return ErrInvalidSeat
	} else if t.isSeated(p) {
		return ErrAlreadySeated
	} else if _, occupied := t.players[seat]; occupied {
		return ErrSeatOccupied
	}

	// if chips < t.MinBuyin() || chips > t.MaxBuyin() {
	// 	return ErrInvalidBuyin
	// }
	// 只限制最小带入。在 room 中做限制，在这里限制会影响到 MTT 比赛的并桌。
	//if chips < t.MinBuyin() {
	//	return ErrInvalidBuyin
	//}

	t.Lock()
	t.players[seat] = &PlayerState{
		player:     p,
		holeCards:  []*HoleCard{},
		chips:      chips,
		beginChips: chips,
		straddle:   straddle,
	}
	t.Unlock()
	return nil
}

func (t *Table) EmptySeats() []int {
	seats := []int{}
	seat := 0
	for seat < t.NumOfSeats() {
		isEmpty := true
		t.RLock()
		for s, _ := range t.players {
			if seat == s {
				isEmpty = false
				break
			}
		}
		t.RUnlock()
		if isEmpty {
			seats = append(seats, seat)
		}
		seat++
	}
	return seats
}

// 坐下的最小带入
func (t *Table) MinBuyin() int {
	return (t.opts.Stakes.SmallBet + t.opts.Stakes.Ante)
}

// 坐下的最大带入
func (t *Table) MaxBuyin() int {
	return (t.opts.Stakes.BigBet * 1000000)
}

// 补充记分牌
func (t *Table) AddChips(seat, chips int) {
	t.Lock()
	defer t.Unlock()
	t.players[seat].chips += chips
}

// 将指定座位的玩家筹码置为0，一手结束时让玩家站起。
func (t *Table) ResetChips(seat int) {
	t.Lock()
	defer t.Unlock()
	t.players[seat].chips = 0
}

func (t *Table) MinPlayChips() int {
	return 1
}

func (t *Table) Straddle() bool {
	return t.opts.Stakes.Straddle
}

// Stand removes the player from the table.  If the player isn't
// seated the command is ignored.
func (t *Table) Stand(p Player) {
	t.Lock()
	defer t.Unlock()
	for seat, pl := range t.players {
		if pl.player.ID() == p.ID() {
			delete(t.players, seat)
			return
		}
	}
}

type tableJSON struct {
	Options      Config                  `json:"options" bson:"options"`
	Deck         *hand.Deck              `json:"deck" bson:"deck"`
	Button       int                     `json:"button" bson:"button"`
	Action       int                     `json:"action" bson:"action"`
	Round        int                     `json:"round" bson:"round"`
	MinRaise     int                     `json:"minRaise" bson:"minRaise"`
	Board        []*hand.Card            `json:"board" bson:"board"`
	Players      map[string]*PlayerState `json:"players" bson:"players"`
	Pot          *Pot                    `json:"pot" bson:"pot"`
	SidePots     []*Pot                  `json:"sidePots" bson:"sidePots"`
	StartedHand  bool                    `json:"startedHand" bson:"startedHand"`
	SmallBetSeat int                     `json:"smallBetSeat" bson:"smallBetSeat"`
	BigBetSeat   int                     `json:"bigBetSeat" bson:"bigBetSeat"`
	UtgSeat      int                     `json:"utgSeat" bson:"utgSeat"`
}

// MarshalJSON implements the json.Marshaler interface.
func (t *Table) MarshalJSON() ([]byte, error) {
	players := map[string]*PlayerState{}
	for seat, player := range t.Players() {
		players[strconv.FormatInt(int64(seat), 10)] = player
	}

	tJSON := &tableJSON{
		Options:      t.opts,
		Deck:         t.deck,
		Button:       t.Button(),
		Action:       t.Action(),
		Round:        t.Round(),
		MinRaise:     t.MinRaise(),
		Board:        t.Board(),
		Players:      players,
		Pot:          t.Pot(),
		SidePots:     t.sidePots,
		StartedHand:  t.startedHand,
		SmallBetSeat: t.smallBetSeat,
		BigBetSeat:   t.bigBetSeat,
		UtgSeat:      t.utgSeat,
	}
	return json.Marshal(tJSON)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *Table) UnmarshalJSON(b []byte) error {
	tJSON := &tableJSON{}
	if err := json.Unmarshal(b, tJSON); err != nil {
		return err
	}

	players := map[int]*PlayerState{}
	for seat, player := range tJSON.Players {
		i, err := strconv.ParseInt(seat, 10, 64)
		if err != nil {
			return err
		}
		players[int(i)] = player
	}

	t.opts = tJSON.Options
	t.dealer = hand.NewDealer()
	t.deck = tJSON.Deck
	t.button = tJSON.Button
	t.action = tJSON.Action
	t.round = tJSON.Round
	t.minRaise = tJSON.MinRaise
	t.board = tJSON.Board
	t.players = players
	t.pot = tJSON.Pot
	t.sidePots = tJSON.SidePots
	t.startedHand = tJSON.StartedHand
	t.smallBetSeat = tJSON.SmallBetSeat
	t.bigBetSeat = tJSON.BigBetSeat
	t.utgSeat = tJSON.UtgSeat

	return nil
}

func (t *Table) setUpHand() {
	t.deck = t.dealer.Deck()
	t.round = 0
	t.button = t.nextSeat(t.button+1, false)
	t.action = -1
	t.pot = newPot(t.NumOfSeats())
	t.straddleSeats = []*StraddleSeat{}

	// reset cards
	t.board = []*hand.Card{}
	t.Lock()
	for _, player := range t.players {
		player.holeCards = []*HoleCard{}
		player.out = false
		player.allin = false
		// set beginChips
		player.beginChips = player.chips
	}
	t.Unlock()
}

func (t *Table) updatePots() {
	t.sidePots = t.Pot().SidePots(t.GetPlayerBeginChips())
}

func (t *Table) setUpRound() {
	// 这里不能加锁，加了会引发死锁。

	t.updatePots()

	// deal board cards
	bCards := t.game().BoardCards(t.deck, round(t.round))
	t.board = append(t.board, bCards...)
	t.resetActed()

	relativePos := t.game().RoundStartSeat(t.HoleCards(), round(t.round))
	for seat, player := range t.players {
		// add hole cards
		hCards := t.game().HoleCards(t.deck, round(t.round))
		player.holeCards = append(player.holeCards, hCards...)

		// add forced bets
		pos := t.relativePosition(seat)
		chips := t.game().ForcedBet(t.HoleCards(), t.opts, round(t.round), seat, pos)

		// set sb/bb/utg seat
		t.setBlindSeat(seat, pos)
		if relativePos == pos {
			t.action = t.nextSeat(seat, true)
			if round(t.round) == preflop {
				t.utgSeat = t.action
			}
		}

		// 说明:mtt比赛玩家筹码小于大小盲也可以继续玩
		if chips > player.chips {
			chips = player.chips
		}
		t.addToPot(seat, chips)
		player.addToPot(chips, t.opts.Stakes.Ante, t.round)
	}

	// reset min raise amounts
	t.minRaise = 0
	t.resetCanRaise(-1)

	// force straddle bet
	if round(t.round) == preflop && t.IsStraddleValid() {
		if state := t.Player(t.utgSeat); state != nil {
			state.SetStraddle(true)
		}
		t.ForceStraddleBet()
	}

	// if everyone is all in or out,  skip round
	count := 0
	for _, player := range t.players {
		if !player.allin && !player.out {
			count++
		}
	}
	if count < 2 {
		t.action = -1
	}
}

func (t *Table) payoutResults(resultsMap map[int][]*Result) {
	t.Lock()
	defer t.Unlock()
	for seat, results := range resultsMap {
		for _, result := range results {
			amount := t.players[seat].chips + result.Chips
			p := t.players[seat]
			p.chips = amount
			t.players[seat] = p
		}
	}
}

func (t *Table) ShowBoardCards(r int) (cards []*hand.Card) {
	if r < t.round {
		return
	}
	switch round(r) {
	case flop:
		return t.game().ShowBoardCards(t.deck, 0, 3)
	case turn:
		return t.game().ShowBoardCards(t.deck, 3, 4)
	case river:
		return t.game().ShowBoardCards(t.deck, 4, 5)
	}
	return
}

func (t *Table) ValidPlayerAction(id int64, a Action, chips int) bool {
	current := t.CurrentPlayer()
	if current.Player().ID() != id {
		return false
	}

	// validate action
	validAction := false
	for _, va := range t.ValidActions() {
		validAction = validAction || va == a
	}

	if !validAction {
		return false
	}

	return true
}

func (t *Table) handleAction(seat int, p *PlayerState, a Action, chips int, timeout bool) error {
	// validate action
	validAction := false
	for _, va := range t.ValidActions() {
		validAction = validAction || va == a
	}
	if !validAction {
		return ErrInvalidAction
	}

	// check if bet or raise amount is invalid
	if (a == Bet || a == Raise) && (chips < t.MinRaise() || chips > t.MaxRaise()) {
		switch a {
		case Bet:
			return ErrInvalidBet
		case Raise:
			return ErrInvalidRaise
		}
	}

	// commit action
	switch a {
	case Fold:
		p.out = true
	case Call:
		outstanding := t.Outstanding()
		if outstanding > p.chips {
			p.addToPot(p.chips, 0, t.round)
		} else {
			p.addToPot(outstanding, 0, t.round)
		}
		t.addToPot(seat, outstanding)
	case Bet:
		betChips := chips - p.roundPot
		p.addToPot(betChips, 0, t.round)
		t.addToPot(seat, betChips)
		t.resetActed()
		if betChips >= t.minRaise {
			t.resetCanRaise(seat)
			t.minRaise = betChips
		}
	case Raise:
		raiseChips := chips - p.roundPot
		if (raiseChips - t.Outstanding()) >= t.minRaise {
			t.resetCanRaise(seat)
			t.minRaise = raiseChips - t.Outstanding()
		}
		p.addToPot(raiseChips, 0, t.round)
		t.addToPot(seat, raiseChips)
		t.resetActed()
	}
	p.canRaise = false
	p.acted = true
	countAllin, countRich := t.countState()
	if t.isNobodyCanPlay() && (countAllin > 1 || (countAllin == 1 && countRich > 0)) {
		t.showdown = true
		t.updatePots()
		t.showHoleCards()
	}

	player := p.Player()
	roundPot := p.roundPot
	if round(t.round) == preflop {
		roundPot = p.pot
	}
	playerAction := PlayerAction{
		PlayerId:   player.ID(),
		Action:     a,
		Chips:      chips,
		ActionTime: time.Now().UTC(),
		Timeout:    timeout,
		RoundPot:   roundPot,
		Pot:        p.pot,
	}
	player.SaveAction(t.Round(), playerAction)
	return nil
}

// 将 show 牌玩家的底牌状态置为显示
func (t *Table) showHoleCards() {
	t.RLock()
	defer t.RUnlock()
	count := 0
	for _, player := range t.players {
		if !player.out {
			count++
		}
	}

	if count > 1 {
		for _, player := range t.players {
			if !player.out {
				for _, card := range player.holeCards {
					card.ExposedCard()
				}
			}
		}
	}
}

// 自动埋牌, seat表示最后一位raise或者allin或者小盲的位置
func (t *Table) AutoConcealedHoleCards(seat int, results map[int][]*Result) []int {
	t.RLock()
	defer t.RUnlock()

	concealSeats := []int{}
	// 座位无效
	if !t.validSeat(seat) {
		return concealSeats
	}
	// 少于2位玩家摊牌
	count := 0
	for _, player := range t.players {
		if !player.out {
			count++
		}
	}
	if count <= 1 {
		return concealSeats
	}
	// 从开始秀牌玩家排序
	seatMap := make(map[int64]int)
	array := []*PlayerState{}
	for i := seat; i < t.NumOfSeats(); i++ {
		p, ok := t.players[i]
		if ok && !p.out {
			array = append(array, p)
			seatMap[p.player.ID()] = i
		}
	}
	for i := 0; i < seat; i++ {
		p, ok := t.players[i]
		if ok && !p.out {
			array = append(array, p)
			seatMap[p.player.ID()] = i
		}
	}
	// 手牌比前位秀牌玩家的弱自动埋牌
	handCard := cardsFromHoleCards(array[0].holeCards)
	target := t.game().FormHighHand(handCard, t.board)
	for _, p := range array {
		handCard = cardsFromHoleCards(p.holeCards)
		highHand := t.game().FormHighHand(handCard, t.board)
		if highHand.CompareTo(target) < 0 {
			for _, card := range p.holeCards {
				card.Visibility = Concealed
			}
			if s, ok := seatMap[p.player.ID()]; ok {
				concealSeats = append(concealSeats, s)
			}
		} else {
			target = highHand
		}
	}
	// expose赢家手牌,包括边池赢家
	newArray := concealSeats
	concealSeats = []int{}
	for _, s := range newArray {
		if _, ok := results[s]; ok {
			p, ok := t.players[s]
			if ok && !p.out {
				for _, card := range p.holeCards {
					card.ExposedCard()
				}
			}
		} else {
			concealSeats = append(concealSeats, s)
		}
	}
	return concealSeats
}

func (t *Table) countState() (countAllin, countRich int) {
	t.RLock()
	defer t.RUnlock()
	for _, player := range t.players {
		if player.allin {
			countAllin++
		}
		if player.allin == false && player.out == false && player.acted {
			countRich++
		}
	}
	return
}

func (t *Table) addToPot(seat, chips int) {
	p := t.Player(seat)
	if chips >= p.chips {
		chips = p.chips
		p.allin = true
	}
	p.chips -= chips
	t.pot.contribute(seat, chips)
}

func (t *Table) nextSeat(seat int, playing bool) int {
	count := 0
	seat = seat % t.NumOfSeats()
	for count < t.NumOfSeats() {
		t.RLock()
		p, ok := t.players[seat]
		t.RUnlock()
		if ok && (!playing || (!p.out && !p.allin && !p.acted)) {
			return seat
		}
		count++
		seat = (seat + 1) % t.NumOfSeats()
	}
	return -1
}

func (t *Table) hasNextHand() bool {
	t.RLock()
	defer t.RUnlock()
	count := 0
	for _, player := range t.players {
		if player.chips > 0 {
			count++
		}
	}
	return count > 1
}

func (t *Table) isSeated(p Player) bool {
	t.RLock()
	defer t.RUnlock()
	for _, pl := range t.players {
		if p.ID() == pl.player.ID() {
			return true
		}
	}
	return false
}

func (t *Table) validSeat(seat int) bool {
	return seat >= 0 && seat < t.NumOfSeats()
}

func (t *Table) relativePosition(seat int) int {
	current := t.button
	count := 0
	for {
		if current == seat {
			break
		}
		current = t.nextSeat(current+1, false)
		count++
	}
	return count
}

func (t *Table) HoleCards() map[int][]*HoleCard {
	t.RLock()
	defer t.RUnlock()
	hCards := map[int][]*HoleCard{}
	for seat, player := range t.players {
		hCards[seat] = player.holeCards
	}
	return hCards
}

func (t *Table) HoleCardsBySeats(seats []int) map[int][]*HoleCard {
	t.RLock()
	defer t.RUnlock()
	hCards := map[int][]*HoleCard{}
	for seat, player := range t.players {
		for _, s := range seats {
			if seat == s {
				hCards[seat] = player.holeCards
			}
		}
	}
	return hCards
}

func (t *Table) GetLeadingPlayer(holeCards map[int][]*hand.Card) Hands {
	highHands := newHands(holeCards, t.board, t.game().FormHighHand)
	lowHands := newHands(holeCards, t.board, t.game().FormLowHand)
	sideHighHands := highHands.handsForSeats(t.pot.seats())
	sideLowHands := lowHands.handsForSeats(t.pot.seats())

	sorting := t.game().Sorting()
	split := len(sideLowHands) > 0
	if !split {
		winners := sideHighHands.winningHands(sorting)
		return winners
	}

	highWinners := sideHighHands.winningHandsForHoldem(t, hand.SortingHigh)
	lowWinners := sideLowHands.winningHandsForHoldem(t, hand.SortingLow)

	if len(lowWinners) == 0 {
		return highWinners
	}
	return highWinners

	// TODO
	//highResults := map[int][]*Result{}
	//lowResults := map[int][]*Result{}
}

func (t *Table) LeadingPlayer() Hands {
	holeCards := cardsFromHoleCardMap(t.HoleCards())
	return t.GetLeadingPlayer(holeCards)
}

// 至少有一个玩家 allin 的最大底池领先的玩家
func (t *Table) MaxPotLeadingPlayerForInsurance() Hands {
	maxPot := &Pot{}
	for _, pot := range t.sidePots {
		if len(pot.contributions) <= 1 {
			continue
		}

		totalAllinPlayer := 0
		for seat, _ := range pot.contributions {
			player := t.Player(seat)
			if player.AllIn() {
				totalAllinPlayer += 1
			}
		}

		if totalAllinPlayer > 0 && pot.Chips() > maxPot.Chips() {
			maxPot = pot
		}
	}

	playerSeats := []int{}
	for seat, _ := range maxPot.contributions {
		playerSeats = append(playerSeats, seat)
	}

	holeCards := cardsFromHoleCardMap(t.HoleCardsBySeats(playerSeats))

	return t.GetLeadingPlayer(holeCards)
}

// 计算保险 outs
// leadingHoleCards: 领先玩家手牌
// backwardHoleCards: 落后玩家手牌
// board: 公共牌
func CalcOuts(leadingHoleCards []*hand.Card, backwardHoleCards [][]*hand.Card, board []*hand.Card) (outs []*hand.Card) {
	// 不计入 OUTS 的牌
	excludedCards := []*hand.Card{}
	excludedCards = append(excludedCards, leadingHoleCards...)
	excludedCards = append(excludedCards, board...)
	for _, item := range backwardHoleCards {
		excludedCards = append(excludedCards, item...)
	}

	cards := hand.CardsOrderByRank()
	isOmaha := false
	if len(leadingHoleCards) == 4 {
		isOmaha = true
	}
	opts := func(c *hand.Config) {}

	calcOuts := []*hand.Card{}

	for _, card := range cards {
		excluded := false
		for _, eCard := range excludedCards {
			if card.Rank() == eCard.Rank() && card.Suit() == eCard.Suit() {
				excluded = true
				break
			}

		}
		if excluded {
			continue
		}

		for _, backward := range backwardHoleCards {
			if isOmaha {
				boardCards := board
				boardCards = append(boardCards, card)
				newBackwardhands := omahaHands(backward, boardCards, opts)
				newBackwardhands = hand.Sort(hand.SortingHigh, hand.DESC, newBackwardhands...)
				newLeadingHands := omahaHands(leadingHoleCards, boardCards, opts)
				newLeadingHands = hand.Sort(hand.SortingHigh, hand.DESC, newLeadingHands...)

				if newBackwardhands[0].CompareTo(newLeadingHands[0]) > 0 {
					calcOuts = append(calcOuts, card)
				}

			} else {
				newBackwardCards := []*hand.Card{card}
				newBackwardCards = append(newBackwardCards, board...)
				newBackwardCards = append(newBackwardCards, backward...)
				newBackwardHand := hand.New(newBackwardCards)

				newLeadingCards := []*hand.Card{card}
				newLeadingCards = append(newLeadingCards, board...)
				newLeadingCards = append(newLeadingCards, leadingHoleCards...)
				newLeadingHand := hand.New(newLeadingCards)

				compareTo := newBackwardHand.CompareTo(newLeadingHand)
				if compareTo > 0 {
					calcOuts = append(calcOuts, card)
				}
			}
		}
	}

	// 对 calcOuts 去重
	for i := 0; i < len(calcOuts); i++ {
		exists := false
		for v := 0; v < i; v++ {
			if calcOuts[v] == calcOuts[i] {
				exists = true
				break
			}
		}
		if !exists {
			outs = append(outs, calcOuts[i])
		}
	}

	return
}

func (t *Table) resetActed() {
	t.Lock()
	defer t.Unlock()
	for _, player := range t.players {
		player.acted = false
	}
}

func (t *Table) resetCanRaise(seat int) {
	t.Lock()
	defer t.Unlock()
	for s, player := range t.players {
		player.canRaise = !(s == seat)
	}
}

func (t *Table) EveryoneFolded() bool {
	count := 0
	for _, player := range t.Players() {
		if !player.out && !player.stand {
			count++
		}
	}
	return count < 2
}

func (t *Table) isNobodyCanPlay() bool {
	t.RLock()
	defer t.RUnlock()
	count := 0
	actedCount := 0
	total := len(t.players)
	for _, player := range t.players {
		if !player.allin && !player.out {
			count++
		}
		if player.acted || player.out || player.allin {
			actedCount++
		}
	}
	return count < 2 && actedCount == total
}

func (t *Table) game() game {
	return t.opts.Game.get()
}

// 设置盲注位置
func (t *Table) setBlindSeat(seat, pos int) {
	if round(t.round) != preflop {
		return
	}

	if len(t.HoleCards()) == 2 {
		switch pos {
		case 0:
			t.smallBetSeat = seat
		case 1:
			t.bigBetSeat = seat
		}
	} else {
		switch pos {
		case 1:
			t.smallBetSeat = seat
		case 2:
			t.bigBetSeat = seat
		}
	}
}

// straddle强制下注
func (t *Table) ForceStraddleBet() {
	// 1xStraddle
	straddleSeat := t.utgSeat
	if ok := t.doOneStraddleBet(straddleSeat, Straddle1); !ok {
		return
	}

	// 2xStraddle
	straddleSeat = t.nextSeat(straddleSeat+1, true)
	if ok := t.doOneStraddleBet(straddleSeat, Straddle2); !ok {
		return
	}

	// 3xStraddle
	straddleSeat = t.nextSeat(straddleSeat+1, true)
	if ok := t.doOneStraddleBet(straddleSeat, Straddle3); !ok {
		return
	}
}

// straddle位下注
func (t *Table) doOneStraddleBet(seat int, category StraddleCategory) bool {
	state := t.Player(seat)
	if state == nil {
		log.WithFields(log.Fields{
			"seat":     seat,
			"tbl":      t,
			"players":  t.players,
			"category": category,
		}).Warning("doOneStraddleBet warn: player not found")
		return false
	}
	if !state.straddle {
		//log.WithFields(log.Fields{
		//	"state":    state,
		//	"seat":     seat,
		//	"category": category,
		//}).Warning("doOneStraddleBet warn: Straddle is not voluntary")
		return false
	}

	betChips := 0
	switch category {
	case Straddle1:
		betChips = 2 * t.BigBet()
	case Straddle2:
		betChips = 2 * 2 * t.BigBet()
	case Straddle3:
		betChips = 2 * 2 * 2 * t.BigBet()
	}
	t.minRaise = betChips / 2
	if betChips > state.chips {
		betChips = state.chips
	}
	t.addToPot(seat, betChips)
	state.addToPot(betChips, 0, t.round)
	t.action = t.nextSeat(seat+1, true)

	tmp := new(StraddleSeat)
	tmp.UserId = state.player.ID()
	tmp.Seat = seat
	tmp.Category = category
	tmp.Voluntary = state.straddle
	t.straddleSeats = append(t.straddleSeats, tmp)
	state.straddle = false

	return true
}

// 重置自愿straddle标记
func (t *Table) ResetPlayerStraddle() {
	t.RLock()
	defer t.RUnlock()
	for _, state := range t.players {
		state.straddle = false
	}
}

func (t *Table) CurrentStraddleSeat() []*StraddleSeat {
	return t.straddleSeats
}

// 当前手已下注straddle标记
func (t *Table) CurrentStraddleSeatStr() string {
	result := string([]byte("[]"))
	if len(t.straddleSeats) == 0 {
		return result
	}
	if view, err := json.Marshal(t.straddleSeats); err == nil {
		result = string(view)
	} else {
		log.WithFields(log.Fields{
			"err":       err,
			"straddles": t.straddleSeats,
			"tbl":       t,
		}).Error("CurrentStraddleSeatStr error: json.Marshal fail")
	}

	return result
}

// straddle是否生效
func (t *Table) IsStraddleValid() bool {
	if !t.Straddle() {
		return false
	}
	count := 0
	for _, p := range t.Players() {
		if !p.stand {
			count++
		}
	}
	return count >= 4
}

func (t *Table) StartedHand() bool {
	return t.startedHand
}

func isNil(o interface{}) bool {
	return o == nil || !reflect.ValueOf(o).Elem().IsValid()
}

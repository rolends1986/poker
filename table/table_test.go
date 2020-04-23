package table_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"poker/hand"
	"poker/pokertest"
	"poker/table"
)

func register() {
	table.RegisterPlayer(Player(0, []PlayerAction{}))
}

type PlayerAction struct {
	Action table.Action
	Chips  int
}

func Player(id int64, actions []PlayerAction) *TestPlayer {
	return &TestPlayer{id: id, actions: actions, index: 0}
}

func HostedPlayer(id int64, tbl *table.Table) *TestPlayer {
	return &TestPlayer{id: id, actions: []PlayerAction{}, hosted: true, tbl: tbl}
}

type TestPlayer struct {
	id       int64
	nickname string
	avatar   string
	country  string
	actions  []PlayerAction
	index    int
	stand    bool
	hosted   bool
	tbl      *table.Table
}

func (p *TestPlayer) Check() {
	p.actions = append(p.actions, PlayerAction{table.Check, 0})
}

func (p *TestPlayer) Call() {
	p.actions = append(p.actions, PlayerAction{table.Call, 0})
}

func (p *TestPlayer) Fold() {
	p.actions = append(p.actions, PlayerAction{table.Fold, 0})
}

func (p *TestPlayer) Bet(amount int) {
	p.actions = append(p.actions, PlayerAction{table.Bet, amount})
}

func (p *TestPlayer) Raise(amount int) {
	p.actions = append(p.actions, PlayerAction{table.Raise, amount})
}

func (p *TestPlayer) ID() int64 {
	return p.id
}

func (p *TestPlayer) Nickname() string {
	return p.nickname
}

func (p *TestPlayer) Country() string {
	return p.country
}

func (p *TestPlayer) Stand() bool {
	return p.stand
}

func (p *TestPlayer) Hosted() bool {
	return p.hosted
}

func (p *TestPlayer) PlayDuration() int64 {
	return 0
}

func (p *TestPlayer) FromID(id int64) (table.Player, error) {
	return Player(id, []PlayerAction{}), nil
}

// 托管操作
func (p *TestPlayer) hostedAction() (action table.Action) {
	for i, a := range p.tbl.ValidActions() {
		if i != 0 {
			if a != table.Check && a != table.Call {
				return table.Fold
			} else {
				return a
			}
		}
	}
	return table.Fold
}

func (p *TestPlayer) Action() (a table.Action, chips int, timeout bool, ignore bool) {
	if p.hosted {
		return p.hostedAction(), 0, false, false
	} else {
		if p.index >= len(p.actions) {
			panic("player " + strconv.FormatInt(p.id, 10) + " doesn't have enough actions")
		}
		a = p.actions[p.index].Action
		chips = p.actions[p.index].Chips
		p.index++
	}
	return
}

func (p *TestPlayer) SaveAction(round int, playerAction table.PlayerAction) {
	p.actions = append(p.actions, PlayerAction{Action: playerAction.Action, Chips: playerAction.Chips})
}

func TestToAndFronJSON(t *testing.T) {
	t.Parallel()
	register()

	// create table
	opts := table.Config{
		Game: table.Holdem,
		Stakes: table.Stakes{
			SmallBet: 1,
			BigBet:   2,
			Ante:     0,
		},
		NumOfSeats: 6,
		Limit:      table.NoLimit,
	}
	p1 := Player(1, []PlayerAction{})
	tbl := table.New(opts, hand.NewDealer())
	if err := tbl.Sit(p1, 0, 100, false); err != nil {
		t.Fatal(err)
	}

	// marshal into json
	b, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	// unmarshal from json
	tblCopy := &table.Table{}
	if err := json.Unmarshal(b, tblCopy); err != nil {
		t.Fatal(err)
	}

	if tblCopy.Opts().Game != table.Holdem {
		t.Error("Unmarshal opts.Game error")
	}

	// marshal back to view
	b, err = json.Marshal(tblCopy)
	if err != nil {
		t.Fatal(err)
	}

	if len(tblCopy.Players()) != 1 {
		t.Fatal("players didn't deserialize correctly")
	}

}

func TestSeating(t *testing.T) {
	t.Parallel()

	opts := table.Config{
		Game: table.Holdem,
		Stakes: table.Stakes{
			SmallBet: 1,
			BigBet:   2,
			Ante:     0,
		},
		NumOfSeats: 6,
	}

	p1 := Player(1, []PlayerAction{})
	p1Dup := Player(1, []PlayerAction{})
	p2 := Player(2, []PlayerAction{})

	tbl := table.New(opts, hand.NewDealer())

	// sit player 1
	if err := tbl.Sit(p1, 0, 100, false); err != nil {
		t.Fatal(err)
	}

	// can't sit dup player 1

	if err := tbl.Sit(p1Dup, 1, 100, false); err != table.ErrAlreadySeated {
		t.Fatal("should already be seated")
	}

	// can't sit player 2 in invalid seat
	if err := tbl.Sit(p2, 6, 100, false); err != table.ErrInvalidSeat {
		t.Fatal("can't sit in invalid seat")
	}

	// can't sit player 2 in occupied seat
	if err := tbl.Sit(p2, 0, 100, false); err != table.ErrSeatOccupied {
		t.Fatal("can't sit in occupied seat")
	}
}

func TestTable_EmptySeats(t *testing.T) {
	t.Parallel()

	opts := table.Config{
		Game: table.Holdem,
		Stakes: table.Stakes{
			SmallBet: 1,
			BigBet:   2,
			Ante:     0,
		},
		NumOfSeats: 6,
	}

	p1 := Player(1, []PlayerAction{})
	p2 := Player(2, []PlayerAction{})

	tbl := table.New(opts, hand.NewDealer())

	seats := tbl.EmptySeats()
	if len(seats) != 6 {
		t.Fatal("test table.EmptySeat() error")
	}
	//t.Log("6", seats)

	// sit player 1
	if err := tbl.Sit(p1, 0, 100, false); err != nil {
		t.Fatal(err)
	}
	seats = tbl.EmptySeats()
	if len(seats) != 5 {
		t.Fatal("test table.EmptySeat() error")
	}
	//t.Log("5", seats)

	// sit player 2
	if err := tbl.Sit(p2, 5, 100, false); err != nil {
		t.Fatal(err)
	}
	seats = tbl.EmptySeats()
	if len(seats) != 4 {
		t.Fatal("test table.EmptySeat() error")
	}
	//t.Log("4", seats)
}

func TestRaises(t *testing.T) {
	t.Parallel()

	opts := table.Config{
		Game: table.Holdem,
		Stakes: table.Stakes{
			SmallBet: 1,
			BigBet:   2,
			Ante:     0,
		},
		NumOfSeats: 6,
	}

	p1 := Player(1, []PlayerAction{})
	p2 := Player(2, []PlayerAction{})
	p3 := Player(3, []PlayerAction{})
	p4 := Player(4, []PlayerAction{})

	tbl := table.New(opts, hand.NewDealer())

	if err := tbl.Sit(p1, 0, 50, false); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Sit(p2, 1, 100, false); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Sit(p3, 2, 52, false); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Sit(p4, 3, 60, false); err != nil {
		t.Fatal(err)
	}

	// preflop
	p1.Call()
	p2.Call()
	p3.Call()
	p4.Check()

	// flop
	p3.Check()
	p4.Check()
	p1.Bet(48)
	p2.Call()
	p3.Raise(2)
	p4.Raise(8)

	for i := 0; i < 12; i++ {
		if _, _, err := tbl.Next(); err != nil {
			t.Fatal(err)
		}
	}

	if tbl.Action() != 1 {
		t.Fatal("action should be on player 2")
	}

	players := tbl.Players()
	if players[1].CanRaise() {
		t.Fatal("player 2 shouldn't be able to raise")
	}

	p2.Call()
	_, _, err := tbl.Next()
	_, _, err = tbl.Next()
	if err != nil {
		t.Fatal(err)
	}
}

func TestStraddle(t *testing.T) {
	// 新建牌桌
	opts := table.Config{
		Game: table.Holdem,
		Stakes: table.Stakes{
			SmallBet: 5,
			BigBet:   10,
			Ante:     5,
			Straddle: true,
		},
		NumOfSeats: 9,
	}
	tbl := table.New(opts, hand.NewDealer())

	// 创建玩家
	p1 := Player(1, []PlayerAction{})
	p2 := Player(2, []PlayerAction{})
	p3 := Player(3, []PlayerAction{})
	p4 := Player(4, []PlayerAction{})
	p5 := Player(5, []PlayerAction{})
	p6 := Player(6, []PlayerAction{})
	if err := tbl.Sit(p1, 0, 100, true); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Sit(p2, 1, 100, true); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Sit(p3, 2, 100, true); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Sit(p4, 3, 100, true); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Sit(p5, 4, 100, true); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Sit(p6, 5, 100, true); err != nil {
		t.Fatal(err)
	}

	if !tbl.IsStraddleValid() {
		t.Error("straddle is not valid")
	}

	tbl.ForceStraddleBet()
	for seat, state := range tbl.Players() {
		t.Log(seat, state)
	}
	result := tbl.CurrentStraddleSeatStr()
	t.Log("current straddle:", result)
}

func TestRiseBlinds(t *testing.T) {
	t.Parallel()

	opts := table.Config{
		Game: table.Holdem,
		Stakes: table.Stakes{
			SmallBet: 5,
			BigBet:   10,
			Ante:     0,
		},
		NumOfSeats: 6,
	}

	tbl := table.New(opts, hand.NewDealer())

	p1 := HostedPlayer(1, tbl)
	p2 := HostedPlayer(2, tbl)

	if err := tbl.Sit(p1, 0, 5000, false); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Sit(p2, 1, 10000, false); err != nil {
		t.Fatal(err)
	}

	hands := 0
	for {
		results, done, err := tbl.Next()

		t.Log("table: ", tbl)

		if done {
			t.Log("done")
		}

		if err != nil {
			t.Fatal("next error", err)
		}

		if results != nil {
			t.Log("results:", results)
			hands++
			if hands != 0 {
				stakes := tbl.Opts().Stakes
				// 每一手涨盲
				tbl.RiseBlinds(stakes.SmallBet*2, stakes.BigBet*2)
			}
			if hands == 3 {
				break
			}
		}
	}
}

func TestTable_ShowBoardCards(t *testing.T) {
	t.Parallel()

	opts := table.Config{
		Game: table.Holdem,
		Stakes: table.Stakes{
			SmallBet: 5,
			BigBet:   10,
			Ante:     0,
		},
		NumOfSeats: 6,
	}

	tbl := table.New(opts, hand.NewDealer())
	b, _ := json.Marshal(tbl)

	t.Logf("table: %s", string(b))

	t.Logf("show round 0: %s", tbl.ShowBoardCards(0))
	t.Logf("show round 1: %v", tbl.ShowBoardCards(1))
	t.Logf("show round 2: %v", tbl.ShowBoardCards(2))
	t.Logf("show round 3: %v", tbl.ShowBoardCards(3))
}

type testCalcOuts struct {
	leadingHoleCards  []*hand.Card
	backwardHoleCards [][]*hand.Card
	board             []*hand.Card
	outs              []*hand.Card
}

var calOutsTests = []testCalcOuts{
	{
		pokertest.Cards("Td", "5s"),
		[][]*hand.Card{
			pokertest.Cards("8s", "Qs"),
		},
		pokertest.Cards("Ac", "Ts", "3c", "8h"),
		pokertest.Cards("Qh", "Qc", "Qd", "8c", "8d"),
	},
	{

		pokertest.Cards("5s", "Qc", "3c", "Js"),
		[][]*hand.Card{
			pokertest.Cards("Ac", "4s", "4c", "9c"),
		},
		pokertest.Cards("3d", "5c", "9s"),
		pokertest.Cards("As", "Ah", "Ad", "9h", "9d", "4h", "4d", "2s", "2h", "2c", "2d"),
	},
	{
		pokertest.Cards("8d", "3d", "Jh", "3c"),
		[][]*hand.Card{
			pokertest.Cards("8h", "6c", "Qs", "8s"),
			pokertest.Cards("6s", "Qh", "Js", "3s"),
		},
		pokertest.Cards("Kc", "9d", "6h", "3h"),
		pokertest.Cards("Ts", "Th", "Tc", "Td", "8c", "6d"),
	},
	{
		pokertest.Cards("Ac", "7c"),
		[][]*hand.Card{
			pokertest.Cards("Td", "4h"),
			pokertest.Cards("As", "6d"),
			pokertest.Cards("Qs", "6c"),
		},
		pokertest.Cards("Kd", "Kh", "5s"),
		pokertest.Cards("Qh", "Qc", "Qd", "Ts", "Th", "Tc", "6s", "6h", "4s", "4c", "4d"),
	},
}

func TestCalcOuts(t *testing.T) {
	for _, test := range calOutsTests {
		t.Log("leadingHoleCards: ", test.leadingHoleCards)
		t.Log("backwardHoleCards: ", test.backwardHoleCards)
		t.Log("board: ", test.board)

		outs := table.CalcOuts(test.leadingHoleCards, test.backwardHoleCards, test.board)
		t.Logf("outs: %v", outs)

		o1, _ := json.Marshal(outs)
		o2, _ := json.Marshal(test.outs)
		if string(o1) != string(o2) {
			t.Errorf("calc outs error: %v != %v", outs, test.outs)
		}
	}
}

func TestLeadingPlayer(t *testing.T) {
	t.Parallel()

	opts := table.Config{
		Game: table.OmahaHi,
		Stakes: table.Stakes{
			SmallBet: 1,
			BigBet:   2,
			Ante:     0,
		},
		NumOfSeats: 6,
	}

	cards := pokertest.Cards("Qc", "6s", "Js", "5s", "8c", "5h", "4d", "7d", "4h", "Qh", "6c", "3s", "8s")
	tbl := table.New(opts, pokertest.Dealer(cards))

	//tbl := table.New(opts, hand.NewDealer())

	p1 := Player(1, []PlayerAction{})
	p2 := Player(2, []PlayerAction{})

	if err := tbl.Sit(p1, 0, 50, false); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Sit(p2, 1, 100, false); err != nil {
		t.Fatal(err)
	}

	// preflop
	p2.Raise(100)
	p1.Call()

	for {
		results, done, err := tbl.Next()

		tbl.PrintTable()

		leadingPlayer := tbl.LeadingPlayer()
		fmt.Println("leading player: ", leadingPlayer)
		//fmt.Println("leading player: ")

		if done {
			t.Log("done")
		}

		if err != nil {
			t.Fatal("next error", err)
		}

		if results != nil {
			//t.Log("results:", results)
			tbl.PrintResults(results)
			break
		}
	}
}

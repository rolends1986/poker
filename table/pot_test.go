package table

import (
	"encoding/json"
	"fmt"
	"git.mulansoft.com/poker/hand"
	"git.mulansoft.com/poker/pokertest"
	"testing"
)

var (
	holdemFunc = func(holeCards []*hand.Card, board []*hand.Card) *hand.Hand {
		cards := append(board, holeCards...)
		return hand.New(cards)
	}

	omahaHiFunc = func(holeCards []*hand.Card, board []*hand.Card) *hand.Hand {
		opts := func(c *hand.Config) {}
		hands := omahaHands(holeCards, board, opts)
		hands = hand.Sort(hand.SortingHigh, hand.DESC, hands...)
		return hands[0]
	}

	omahaLoFunc = func(holeCards []*hand.Card, board []*hand.Card) *hand.Hand {
		hands := omahaHands(holeCards, board, hand.AceToFiveLow)
		hands = hand.Sort(hand.SortingLow, hand.DESC, hands...)
		if hands[0].CompareTo(eightOrBetter) <= 0 {
			return hands[0]
		}
		return nil
	}
)

type player struct {
	id       int64
	nickname string
	avatar   string
	country  string
	stand    bool
	hosted   bool
}

func (p *player) ID() int64 {
	return p.id
}

func (p *player) FromID(id int64) (Player, error) {
	return &player{
		id:       p.ID(),
		nickname: "",
		avatar:   "",
		stand:    false,
	}, nil
}

func (p *player) Nickname() string {
	return p.nickname
}

func (p *player) Country() string {
	return p.country
}

func (p *player) Stand() bool {
	return p.stand
}

func (p *player) PlayDuration() int64 {
	return 0
}

func (p *player) Action() (Action, int, bool, bool) {
	return "sit", 1, false, false
}

func (p *player) Hosted() bool {
	return p.hosted
}
func (p *player) SaveAction(round int, playerAction PlayerAction) {

}

type testSidePot struct {
	numOfseats          int         // 牌局座位数
	playerBeginChips    map[int]int // 玩家的带入筹码
	contribute          map[int]int // 玩家的下注筹码
	sidePots            string      // 底池结果
	totalSidePot        int         // 最终的底池总数
	playerTotalSidePots map[int]int // 玩家参与的边池总数
}

var sidePotTests = []testSidePot{
	{
		9,
		map[int]int{0: 5, 1: 20, 2: 100},
		map[int]int{0: 5, 1: 20, 2: 50},
		`[{"contributions":{"0":5,"1":5,"2":5},"chips":15},{"contributions":{"1":15,"2":15},"chips":30},{"contributions":{"2":30},"chips":30}]`,
		3,
		map[int]int{0: 1, 1: 2, 2: 3},
	},
	{
		9,
		map[int]int{0: 100, 1: 100, 2: 100, 4: 100, 5: 100, 6: 100, 8: 100},
		map[int]int{0: 1, 1: 1, 2: 1, 4: 0, 5: 0, 6: 53, 8: 53},
		`[{"contributions":{"0":1,"1":1,"2":1,"6":53,"8":53},"chips":109}]`,
		1,
		map[int]int{0: 1, 1: 1, 2: 1, 4: 0, 5: 0, 6: 1, 8: 1},
	},
	{
		9,
		map[int]int{0: 200, 1: 300, 2: 100, 3: 200},
		map[int]int{0: 2, 1: 222, 2: 100, 3: 200},
		`[{"contributions":{"0":2,"1":100,"2":100,"3":100},"chips":302},{"contributions":{"1":100,"3":100},"chips":200},{"contributions":{"1":22},"chips":22}]`,
		3,
		map[int]int{0: 1, 1: 3, 2: 1, 3: 2},
	},
}

func TestSidePot(t *testing.T) {
	t.Parallel()

	for i, test := range sidePotTests {
		p := newPot(test.numOfseats)
		for seat, chips := range test.contribute {
			p.contribute(seat, chips)
		}

		sidePots := p.SidePots(test.playerBeginChips)

		j, _ := json.Marshal(sidePots)
		sidePotStr := string(j)
		t.Logf("the %v test case: sidePots: %v", i+1, sidePots)
		if sidePotStr != test.sidePots {
			t.Fatalf("the %v test case: sidePosts must be %v, now is: %v.", i+1, test.sidePots, sidePotStr)
		}

		totalSidePot := len(sidePots)
		if totalSidePot != test.totalSidePot {
			t.Fatalf("the %v test case: sidePosts's length must be %v, now is: %v.", i+1, test.totalSidePot, totalSidePot)
		}

		playerTotalSidePots := map[int]int{}
		playerTotalContribute := map[int]int{}
		for _, sp := range sidePots {
			for seat, chips := range sp.contributions {
				playerTotalSidePots[seat] += 1
				playerTotalContribute[seat] += chips
			}
		}

		for seat, chips := range test.playerTotalSidePots {
			if playerTotalSidePots[seat] != chips {
				t.Fatalf("the %v test case: total side pots of seat %v must be %v, now is %v.", i+1, seat, chips, playerTotalSidePots[seat])
			}
			if playerTotalContribute[seat] != test.contribute[seat] {
				t.Fatalf("the %v test case: contribute of seat %v must be %v, now is %v.", i+1, seat, test.contribute[seat], playerTotalContribute[seat])
			}
		}
	}
}

func TestPotJSON(t *testing.T) {
	t.Parallel()

	p := newPot(3)
	p.contribute(0, 1)

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	// unmarshal from json
	p = &Pot{}
	if err := json.Unmarshal(b, p); err != nil {
		t.Fatal(err)
	}
	if p.Chips() != 1 {
		t.Errorf("after json roundtrip pot.Chips() = %v; want %v", p.Chips(), 1)
	}
}

func TestHighPot(t *testing.T) {
	t.Parallel()
	tbl := holdemTable()

	p := newPot(3)
	p.contribute(0, 5)
	p.contribute(1, 10)
	p.contribute(2, 15)

	seatToHoleCards := map[int][]*hand.Card{
		0: []*hand.Card{
			hand.AceSpades,
			hand.AceHearts,
		},
		1: []*hand.Card{
			hand.QueenSpades,
			hand.QueenHearts,
		},
		2: []*hand.Card{
			hand.KingSpades,
			hand.KingHearts,
		},
	}

	board := pokertest.Cards("Ad", "Kd", "Qd", "2d", "2h")
	hands := newHands(seatToHoleCards, board, holdemFunc)
	payout := p.payout(0, tbl, hands, nil, hand.SortingHigh, 0)
	for seat, results := range payout {
		switch seat {
		case 0:
			if len(results) != 1 {
				t.Fatal("seat 0 should win one pot")
			}
		case 1:
			if len(results) != 0 {
				t.Fatal("seat 1 should win no pots")
			}
		case 2:
			if len(results) != 2 {
				t.Fatal("seat 2 should win two pots")
			}
		}
	}
}

func TestHighLowPot(t *testing.T) {
	t.Parallel()
	tbl := omahaHiTable()
	p1 := &player{id: 0}
	p2 := &player{id: 1}
	p3 := &player{id: 2}

	tbl.players[0] = &PlayerState{
		player: p1,
	}
	tbl.players[1] = &PlayerState{
		player: p2,
	}
	tbl.players[2] = &PlayerState{
		player: p3,
	}

	p := newPot(3)
	p.contribute(0, 5)
	p.contribute(1, 5)
	p.contribute(2, 5)

	seatToHoleCards := map[int][]*hand.Card{
		0: []*hand.Card{
			hand.AceHearts,
			hand.TwoClubs,
			hand.SevenDiamonds,
			hand.KingHearts,
		},
		1: []*hand.Card{
			hand.AceDiamonds,
			hand.FourClubs,
			hand.ThreeDiamonds,
			hand.SixSpades,
		},
		2: []*hand.Card{
			hand.AceSpades,
			hand.TwoHearts,
			hand.JackDiamonds,
			hand.JackClubs,
		},
	}

	board := pokertest.Cards("7s", "Kd", "8h", "Jh", "5c")
	highHands := newHands(seatToHoleCards, board, omahaHiFunc)
	lowHands := newHands(seatToHoleCards, board, omahaLoFunc)
	payout := p.payout(0, tbl, highHands, lowHands, hand.SortingHigh, 0)

	if len(payout) < 3 {
		t.Errorf("pot.Payout() should have 3 results")
	}

	for seat, results := range payout {
		fmt.Println("seat:", seat)
		fmt.Println(results)
		switch seat {
		case 0:
			if len(results) != 1 && total(results) != 4 {
				t.Errorf("seat 0 should win 4 chips")
			}
		case 1:
			if len(results) != 1 && total(results) != 8 {
				t.Errorf("seat 1 should win 8 chips")
			}
		case 2:
			if len(results) != 1 && total(results) != 3 {
				t.Errorf("seat 2 should 3 chips")
			}
		}
	}
}

func TestOmahaHiPot(t *testing.T) {
	t.Parallel()
	tbl := omahaHiTable()
	p1 := &player{id: 0}
	p2 := &player{id: 1}
	p3 := &player{id: 2}

	tbl.players[0] = &PlayerState{
		player: p1,
	}
	tbl.players[1] = &PlayerState{
		player: p2,
	}
	tbl.players[2] = &PlayerState{
		player: p3,
	}

	p := newPot(3)
	p.contribute(0, 5)
	p.contribute(1, 5)
	p.contribute(2, 5)

	seatToHoleCards := map[int][]*hand.Card{
		0: []*hand.Card{
			hand.AceClubs,
			hand.AceSpades,
			hand.TenHearts,
			hand.TwoSpades,
		},
		1: []*hand.Card{
			hand.NineHearts,
			hand.JackClubs,
			hand.ThreeDiamonds,
			hand.SevenDiamonds,
		},
		2: []*hand.Card{
			hand.SixDiamonds,
			hand.QueenSpades,
			hand.SevenHearts,
			hand.QueenDiamonds,
		},
	}

	board := pokertest.Cards("8c", "Kc", "Qc", "4c", "5c")
	highHands := newHands(seatToHoleCards, board, omahaHiFunc)
	fmt.Println(highHands)
	payout := p.payout(0, tbl, highHands, nil, hand.SortingHigh, 0)

	// if len(payout) != 2 {
	// 	t.Errorf("pot.Payout() should have 2 results")
	// }

	for seat, results := range payout {
		fmt.Println("seat:", seat)
		fmt.Println(results)
		switch seat {
		case 1:
			if len(results) != 1 && total(results) != 15 {
				t.Errorf("seat 1 should win 15 chips")
			}

		}
	}
}

func total(results []*Result) int {
	chips := 0
	for _, r := range results {
		chips += r.Chips
	}
	return chips
}
func TestMultiWinnerPot(t *testing.T) {
	t.Parallel()
	tbl := holdemTable()

	p := newPot(5)
	p.contribute(0, 5)
	p.contribute(1, 10)
	p.contribute(2, 14)
	p.contribute(3, 15)
	p.contribute(4, 14)

	seatToHoleCards := map[int][]*hand.Card{
		0: []*hand.Card{
			hand.AceSpades,
			hand.QueenHearts,
		},
		1: []*hand.Card{
			hand.QueenSpades,
			hand.AceHearts,
		},
		2: []*hand.Card{
			hand.KingSpades,
			hand.SevenHearts,
		},
		3: []*hand.Card{
			hand.AceDiamonds,
			hand.QueenClubs,
		},
		4: []*hand.Card{
			hand.JackHearts,
			hand.EightClubs,
		},
	}

	board := pokertest.Cards("Ac", "Kd", "Qd", "3c", "2h")
	hands := newHands(seatToHoleCards, board, holdemFunc)
	payout := p.payout(0, tbl, hands, nil, hand.SortingHigh, 2)
	for seat, results := range payout {

		fmt.Println("seat:", seat)
		fmt.Println("results:", results)
	}
}

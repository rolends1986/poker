package table

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"git.mulansoft.com/poker/hand"
	"gopkg.in/mgo.v2/bson"
	"sync"
)

// Results is a mapping of each seat with its slice of results.
type Results map[int][]*Result

// Share is the rights a winner has to the pot.
type Share string

const (
	// WonHigh indicates that the high hand was won.
	WonHigh Share = "WonHigh"

	// WonLow indicates that the low hand was won.
	WonLow Share = "WonLow"

	// SplitHigh indicates that the high hand was split.
	SplitHigh Share = "SplitHigh"

	// SplitLow indicates that the low hand was split.
	SplitLow Share = "SplitLow"
)

// A Result is a player's winning result from a showdown.
type Result struct {
	PotNo int 		 `json:"potNo"`
	Hand  *hand.Hand `json:"hand"`
	Chips int        `json:"chips"`
	Share Share      `json:"share"`
}

// String returns a string useful for debugging.
func (p *Result) String() string {
	const format = "%s for %d chips with %s in %v pot"
	return fmt.Sprintf(format, p.Share, p.Chips, p.Hand, p.PotNo)
}

// MarshalJSON implements the json.Marshaler interface.
// The json format is:
// {"hand": {"ranking":9,"cards":["A♠","K♠","Q♠","J♠","T♠"],"description":"royal flush"}, "chips": 4, "share": "WonHigh"}
func (r *Result) MarshalJSON() ([]byte, error) {
	b, err := r.Hand.MarshalJSON()
	if err != nil {
		return []byte{}, err
	}
	const format = `{"hand":%v,"chips":%v,"share":"%v"}`
	s := fmt.Sprintf(format, string(b), r.Chips, r.Share)
	return []byte(s), nil
}

type ResultJSON struct {
	Hand  hand.HandJSON `json:"hand"`
	Chips int           `json:"chips"`
	Share Share         `json:"share"`
}

func (r *Result) ResultJSON() ResultJSON {
	resultJSON := ResultJSON{
		Chips: r.Chips,
		Share: r.Share,
	}
	if r.Hand != nil {
		resultJSON.Hand = r.Hand.HandJSON()
	}
	return resultJSON
}

// A Pot is the collection of contributions made by players during
// a hand. After the showdown, the pot's chips are divided among the
// winners.
type Pot struct {
	contributions map[int]int
	sync.RWMutex
}

// newPot returns a pot with zero contributions for all seats.
func newPot(numOfSeats int) *Pot {
	contributions := map[int]int{}
	for i := 0; i < numOfSeats; i++ {
		contributions[i] = 0
	}
	return &Pot{contributions: contributions}
}

// String returns a string useful for debugging.
func (p *Pot) String() string {
	const format = "contributions: %v"
	return fmt.Sprintf(format, p.contributions)
}

func (p *Pot) Contributions() map[int]int {
	return p.contributions
}

func (p *Pot) GetContribution(seat int) int {
	p.RLock()
	defer p.RUnlock()
	return p.contributions[seat]
}

// Chips returns the number of chips in the pot.
func (p *Pot) Chips() int {
	chips := 0
	p.RLock()
	for _, c := range p.contributions {
		chips += c
	}
	p.RUnlock()
	return chips
}

// Outstanding returns the amount required for a seat to call the
// largest current bet or raise.
func (p *Pot) Outstanding(seat int) int {
	p.RLock()
	defer p.RUnlock()
	most := 0

	for _, chips := range p.contributions {
		if chips > most {
			most = chips
		}
	}

	return most - p.contributions[seat]
}

// Contribute contributes the chip amount from the seat given
func (p *Pot) contribute(seat, chips int) {
	if chips < 0 {
		panic("table: pot contribute negative bet amount")
	}
	p.Lock()
	p.contributions[seat] += chips
	p.Unlock()
}

// Take creates results with the seat taking the entire pot
func (p *Pot) take(seat int) Results {
	results := map[int][]*Result{
		seat: []*Result{
			{Hand: nil, Chips: p.Chips(), Share: WonHigh},
		},
	}
	return results
}

// payout takes the high and low hands to produce pot results.
// Sorting determines how a non-split pot winning hands are sorted.
func (p *Pot) payout(potNo int,t *Table, highHands, lowHands Hands, sorting hand.Sorting, button int) Results {
	sidePots := p.SidePots(t.GetPlayerBeginChips())
	if len(sidePots) > 1 {
		results := map[int][]*Result{}
		for potNo, sp := range sidePots {
			r := sp.payout(potNo, t, highHands, lowHands, sorting, button)
			results = combineResults(results, r)
		}
		return results
	}

	sideHighHands := highHands.handsForSeats(p.seats())
	sideLowHands := lowHands.handsForSeats(p.seats())

	split := len(sideLowHands) > 0
	if !split {
		winners := sideHighHands.winningHands(sorting)
		switch sorting {
		case hand.SortingHigh:
			return p.resultsFromWinners(potNo, winners, p.Chips(), button, highPotShare)
		case hand.SortingLow:
			return p.resultsFromWinners(potNo, winners, p.Chips(), button, lowPotShare)
		}
	}

	highWinners := sideHighHands.winningHandsForHoldem(t, hand.SortingHigh)
	lowWinners := sideLowHands.winningHandsForHoldem(t, hand.SortingLow)

	if len(lowWinners) == 0 {
		return p.resultsFromWinners(potNo, highWinners, p.Chips(), button, highPotShare)
	}

	highResults := map[int][]*Result{}
	lowResults := map[int][]*Result{}

	highAmount := p.Chips() / 2
	if highAmount%2 == 1 {
		highAmount++
	}

	highResults = p.resultsFromWinners(potNo, highWinners, highAmount, button, highPotShare)
	lowResults = p.resultsFromWinners(potNo, lowWinners, p.Chips()/2, button, lowPotShare)
	return combineResults(highResults, lowResults)
}

// GetBSON implements bson.Getter.
func (p *Pot) GetBSON() (interface{}, error) {
	return p.PotJSON(), nil
}

// SetBSON implements bson.Setter.
func (p *Pot) SetBSON(raw bson.Raw) error {
	pJSON := &PotJSON{}
	if err := raw.Unmarshal(pJSON); err != nil {
		return err
	}

	return nil
}

type PotJSON struct {
	Contributions map[string]int `json:"contributions" bson:"contributions"`
	Chips         int            `json:"chips" bson:"chips"`
}

func (p *Pot) PotJSON() *PotJSON {
	m := map[string]int{}
	p.RLock()
	for seat, chips := range p.contributions {
		seatStr := strconv.FormatInt(int64(seat), 10)
		m[seatStr] = chips
	}
	p.RUnlock()

	j := &PotJSON{
		Contributions: m,
		Chips:         p.Chips(),
	}
	return j
}

// MarshalJSON conforms to the json.Marshaler interface
func (p *Pot) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.PotJSON())
}

// UnmarshalJSON conforms to the json.Marshaler interface
func (p *Pot) UnmarshalJSON(b []byte) error {
	j := &PotJSON{}
	if err := json.Unmarshal(b, j); err != nil {
		return err
	}

	m := map[int]int{}
	for seatStr, chips := range j.Contributions {
		seat, err := strconv.ParseInt(seatStr, 10, 64)
		if err != nil {
			return err
		}
		m[int(seat)] = chips
	}

	p.contributions = m
	return nil
}

// resultsFromWinners forms results for winners of the pot
func (p *Pot) resultsFromWinners(potNo int, winners Hands, chips, button int, f func(n int) Share) map[int][]*Result {
	results := map[int][]*Result{}
	winningSeats := []int{}
	for seat, hand := range winners {
		winningSeats = append(winningSeats, int(seat))
		results[seat] = []*Result{&Result{
			PotNo: potNo,
			Hand:  hand,
			Chips: chips / len(winners),
			Share: f(len(winners)),
		}}
	}
	sort.IntSlice(winningSeats).Sort()

	remainder := chips % len(winners)
	seatToCheck := button % 10
	for {
		if remainder == 0 {
			break
		}
		for _, seat := range winningSeats {
			if seat == seatToCheck {
				results[seat][0].Chips++
				remainder--
				break
			}
		}
		seatToCheck++
		seatToCheck = seatToCheck % 10
	}
	return results
}

// sidePots forms an array of side pots including the main pot
func (p *Pot) SidePots(playerBeginChips map[int]int) []*Pot {
	// get site pot contribution amounts
	amounts := p.sidePotAmounts()
	pots := []*Pot{}
	for i, a := range amounts {
		side := &Pot{
			contributions: map[int]int{},
		}

		last := 0
		if i != 0 {
			last = amounts[i-1]
		}

		for seat, chips := range p.contributions {
			if chips > last && chips >= a {
				side.contributions[seat] = a - last
			} else if chips > last && chips < a {
				side.contributions[seat] = chips - last
			}
		}

		pots = append(pots, side)
	}

	sidePots := []*Pot{}
	lastAllIn := false

	for i, pot := range pots {
		// 判断当前的座位的玩家是否 allin
		hasAllin := false
		for seat, chips := range pot.contributions {
			totalChips := chips
			for _, s := range sidePots {
				totalChips += s.contributions[seat]
			}
			if totalChips == playerBeginChips[seat] {
				hasAllin = true
			}
		}

		if lastAllIn || (lastAllIn && hasAllin) || i == 0 {
			// 以下情况需要产生独立的边池
			// 1. 上个底池有玩家 all in。
			// 2. 在上个底池有玩家 all in 的情况下， 本底池也有玩家 all in，如果本地没有玩家 all in ，是因为弃牌产生的，则并入上个 all in 底池中。
			sidePots = append(sidePots, pot)
		} else {
			last := len(sidePots)-1
			for seat, chips := range pot.contributions {
				sidePots[last].contributions[seat] = sidePots[last].contributions[seat] + chips
			}
		}

		lastAllIn = hasAllin
	}

	return sidePots
}

// sidePotAmounts finds the contribution divisions for side pots
func (p *Pot) sidePotAmounts() []int {
	amounts := []int{}
	p.Lock()
	for seat, chips := range p.contributions {
		if chips == 0 {
			delete(p.contributions, seat)
		} else {
			found := false
			for _, a := range amounts {
				found = found || a == chips
			}
			if !found {
				amounts = append(amounts, chips)
			}
		}
	}
	p.Unlock()
	sort.IntSlice(amounts).Sort()
	return amounts
}

func (p *Pot) seats() []int {
	seats := []int{}
	p.RLock()
	for seat := range p.contributions {
		seats = append(seats, seat)
	}
	p.RUnlock()
	return seats
}

func highPotShare(n int) Share {
	if n == 1 {
		return WonHigh
	}
	return SplitHigh
}

func lowPotShare(n int) Share {
	if n == 1 {
		return WonLow
	}
	return SplitLow
}

func combineResults(results ...map[int][]*Result) map[int][]*Result {
	combined := map[int][]*Result{}
	for _, m := range results {
		for k, v := range m {
			s := append(combined[k], v...)
			combined[k] = s
		}
	}
	return combined
}

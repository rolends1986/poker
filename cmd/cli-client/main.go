package main

import (
	"errors"
	"fmt"
	"game/utils"
	"strconv"
	"strings"

	"github.com/rolends1986/poker/hand"
	"github.com/rolends1986/poker/table"
)

const (
	fold  = "fold"
	check = "check"
	call  = "call"
	bet   = "bet"
	raise = "raise"
)

var (
	tbl            *table.Table
	actionsRecords map[int][]table.PlayerAction
	isAdd          bool
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

func (p *player) Nickname() string {
	return p.nickname
}

func (p *player) Country() string {
	return p.country
}

func (p *player) Stand() bool {
	return p.stand
}

func (p *player) Hosted() bool {
	return p.hosted
}

func (p *player) PlayDuration() int64 {
	return 0
}

func (p *player) FromID(id int64) (table.Player, error) {
	return &player{
		id:       p.ID(),
		nickname: "",
		avatar:   "",
		stand:    false,
	}, nil
}

func (p *player) Action() (table.Action, int, bool, bool) {
	current := tbl.CurrentPlayer()

	// get action from input
	actions := []string{}
	for _, a := range tbl.ValidActions() {
		actions = append(actions, strings.ToLower(string(a)))
	}

	// show info
	currentInfoFormat := "\nChips %d, Outstanding %d, MinRaise %d, MaxRaise %d"
	fmt.Printf(currentInfoFormat, current.Chips(), tbl.Outstanding(), tbl.MinRaise(), tbl.MaxRaise())

	// get action from input
	var input string
	actionFormat := "\nPlayer %v Action (%s):\n"
	fmt.Printf(actionFormat, p.ID(), strings.Join(actions, ","))
	if _, err := fmt.Scan(&input); err != nil {
		fmt.Println("Error", err)
		return p.Action()
	}

	// parse action
	action, err := actionFromInput(input)
	if err != nil {
		fmt.Println("Error", err)
		return p.Action()
	}
	if !(action == table.Bet || action == table.Raise) {
		return action, 0, false, false
	}

	// get amount from input
	amountFormat := "\nEnter Bet / Raise Amount:\n"
	fmt.Printf(amountFormat)
	if _, err := fmt.Scan(&input); err != nil {
		fmt.Println("Error", err)
		return p.Action()
	}

	// parse amount
	chips, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		fmt.Println("Error", err)
		return p.Action()
	}
	return action, int(chips), false, false
}

func (p *player) SaveAction(round int, playerAction table.PlayerAction) {
	actionRecode, ok := actionsRecords[round]
	if ok {
		actionsRecords[round] = append(actionRecode, playerAction)
	} else {
		actionsRecords[round] = []table.PlayerAction{
			playerAction,
		}
	}
}

func main() {
	// p1 := playerFromInput("Player 1")
	// p2 := playerFromInput("Player 2")
	// p3 := playerFromInput("Player 3")
	// p4 := playerFromInput("Player 4")
	// p5 := playerFromInput("Player 5")
	// p6 := playerFromInput("Player 6")

	p1 := &player{id: 1}
	p2 := &player{id: 2}
	p3 := &player{id: 3}

	opts := table.Config{
		Game:       table.Holdem,
		Limit:      table.NoLimit,
		Stakes:     table.Stakes{SmallBet: 1, BigBet: 2, Ante: 0},
		NumOfSeats: 9,
	}
	tbl = table.New(opts, hand.NewDealer())
	actionsRecords = make(map[int][]table.PlayerAction)
	if err := tbl.Sit(p1, 0, 100, false); err != nil {
		panic(err)
	}
	if err := tbl.Sit(p2, 1, 50, false); err != nil {
		panic(err)
	}
	if err := tbl.Sit(p3, 2, 10, false); err != nil {
		panic(err)
	}
	// if err := tbl.Sit(p4, 3, 400); err != nil {
	// 	panic(err)
	// }
	// if err := tbl.Sit(p5, 4, 400); err != nil {
	// 	panic(err)
	// }
	// if err := tbl.Sit(p6, 5, 400); err != nil {
	// 	panic(err)
	// }

	runTable(tbl)
	fmt.Println("DONE")
}

func runTable(tbl *table.Table) {
	for {
		results, done, err := tbl.Next()
		fmt.Printf("\n--------\nshowndown: %v\n----------\n", tbl.Showdown())
		if done {
			return
		}
		printTable(tbl)
		if err != nil {
			fmt.Println("Error", err)
		}
		if results != nil {
			printResults(tbl, results)
			// if !isAdd {
			// 	p6 := playerFromInput("Player 6")
			// 	if err := tbl.Sit(p6, 5, 400); err != nil {
			// 		panic(err)
			// 	}
			// 	isAdd = true
			// }
		}
	}
}

func printTable(tbl *table.Table) {
	players := tbl.Players()
	fmt.Println("")
	fmt.Println("-----Table-----")
	fmt.Println(tbl)
	for key, value := range players {
		fmt.Println(key, value)
	}
	fmt.Printf("side pots: %v\n", tbl.SidePots())
	fmt.Printf("actions records: %v\n", actionsRecords[tbl.Round()])
	fmt.Println("-----Table-----")
	fmt.Println("")
}

func printResults(tbl *table.Table, results map[int][]*table.Result) {
	fmt.Printf("side pots: \n%v\n", tbl.Pot().SidePots(tbl.GetPlayerBeginChips()))
	players := tbl.Players()
	for seat, resultList := range results {
		for _, result := range resultList {
			playerId := utils.Int64ToString(players[seat].Player().ID())
			fmt.Println(playerId+":", result)
		}
	}
}

func playerFromInput(desc string) table.Player {
	var input string
	fmt.Printf("\nPick %s name:\n", desc)
	if _, err := fmt.Scan(&input); err != nil {
		fmt.Println("Error", err)
		return playerFromInput(desc)
	}
	return &player{id: utils.StringToInt64(input)}
}

func actionFromInput(input string) (table.Action, error) {
	switch input {
	case fold:
		return table.Fold, nil
	case check:
		return table.Check, nil
	case call:
		return table.Call, nil
	case bet:
		return table.Bet, nil
	case raise:
		return table.Raise, nil
	}
	return table.Fold, errors.New(input + " is not an action.")
}

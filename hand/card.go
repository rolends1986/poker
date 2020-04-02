package hand

import (
	"errors"
	"strings"
)

// A Rank represents the rank of a card.
type Rank string

const (
	// Two has the rank of 2
	Two Rank = "2"

	// Three has the rank of 3
	Three Rank = "3"

	// Four has the rank of 4
	Four Rank = "4"

	// Five has the rank of 5
	Five Rank = "5"

	// Six has the rank of 6
	Six Rank = "6"

	// Seven has the rank of 7
	Seven Rank = "7"

	// Eight has the rank of 8
	Eight Rank = "8"

	// Nine has the rank of 9
	Nine Rank = "9"

	// Ten has the rank of 10
	Ten Rank = "T"

	// Jack has the rank of J
	Jack Rank = "J"

	// Queen has the rank of Q
	Queen Rank = "Q"

	// King has the rank of K
	King Rank = "K"

	// Ace has the rank of A
	Ace Rank = "A"
)

// IndexOf returns the index of the rank in the ascending order of ranks.
// IndexOf returns -1 if the rank is not found.
func (r Rank) indexOf() int {
	for i, rank := range allRanks() {
		if r == rank {
			return i
		}
	}
	return -1
}

// String returns a string in the format "2"
func (r Rank) String() string {
	return string(r)
}

// singularName returns the name of the rank in singular form such as "two" for Two.
func (r Rank) singularName() string {
	return singularNames[r]
}

// pluralName returns the name of the rank in plural form such as "twos" for Two.
func (r Rank) pluralName() string {
	return pluralNames[r]
}

// Valid returns true if the rank is valid
func (r Rank) valid() bool {
	return r.indexOf() != -1
}

func (r Rank) aceLowIndexOf() int {
	for i, rank := range allAceLowRanks() {
		if r == rank {
			return i
		}
	}
	return -1
}

type byAceHighRank []Rank

func (a byAceHighRank) Len() int { return len(a) }

func (a byAceHighRank) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a byAceHighRank) Less(i, j int) bool {
	iRank, jRank := a[i], a[j]
	iIndex, jIndex := iRank.indexOf(), jRank.indexOf()
	return iIndex < jIndex
}

type byAceLowRank []Rank

func (a byAceLowRank) Len() int { return len(a) }

func (a byAceLowRank) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a byAceLowRank) Less(i, j int) bool {
	iRank, jRank := a[i], a[j]
	iIndex, jIndex := iRank.aceLowIndexOf(), jRank.aceLowIndexOf()
	return iIndex < jIndex
}

var (
	singularNames = map[Rank]string{
		Two:   "two",
		Three: "three",
		Four:  "four",
		Five:  "five",
		Six:   "six",
		Seven: "seven",
		Eight: "eight",
		Nine:  "nine",
		Ten:   "ten",
		Jack:  "jack",
		Queen: "queen",
		King:  "king",
		Ace:   "ace",
	}

	pluralNames = map[Rank]string{
		Two:   "twos",
		Three: "threes",
		Four:  "fours",
		Five:  "fives",
		Six:   "sixes",
		Seven: "sevens",
		Eight: "eights",
		Nine:  "nines",
		Ten:   "tens",
		Jack:  "jacks",
		Queen: "queens",
		King:  "kings",
		Ace:   "aces",
	}
)

// A Suit represents the suit of a card.
type Suit string

const (
	// Spades has a suit of ♠
	Spades Suit = "♠"

	// Hearts has a suit of ♥
	Hearts Suit = "♥"

	// Diamonds has a suit of ♦
	Diamonds Suit = "♦"

	// Clubs has a suit of ♣
	Clubs Suit = "♣"
)

// String returns a string in the format "♠"
func (s Suit) String() string {
	return string(s)
}

func (s Suit) valid() bool {
	return strings.Contains("♠♥♦♣", string(s))
}

// A Card represents a playing card in the game of poker.  It is composed of a rank and suit.
type Card struct {
	TheRank Rank `json:"rank" bson:"rank"`
	TheSuit Suit `json:"suit" bson:"suit"`
}

// Rank returns the rank of the card.
func (c *Card) Rank() Rank {
	return c.TheRank
}

// Suit returns the suit of the card.
func (c *Card) Suit() Suit {
	return c.TheSuit
}

// String returns a string in the format "4♠"
func (c *Card) String() string {
	return string(c.Rank()) + string(c.Suit())
}

// MarshalText implements the encoding.TextMarshaler interface.
// The text format is "4♠".
func (c *Card) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// The card is expected to be in the format "4♠".
func (c *Card) UnmarshalText(text []byte) error {
	var rank Rank
	var suit Suit
	const errStr = `card: serialization should be of the format "4♠"`
	for i, c := range string(text) {
		if i == 0 && Rank(c).valid() {
			rank = Rank(c)
		} else if i == 1 && Suit(c).valid() {
			suit = Suit(c)
		} else {
			return errors.New(errStr)
		}
	}
	if rank == "" || suit == "" {
		return errors.New(errStr)
	}
	c.TheRank = rank
	c.TheSuit = suit
	return nil
}

var (
	AceSpades   = &Card{TheRank: Ace, TheSuit: Spades}
	KingSpades  = &Card{TheRank: King, TheSuit: Spades}
	QueenSpades = &Card{TheRank: Queen, TheSuit: Spades}
	JackSpades  = &Card{TheRank: Jack, TheSuit: Spades}
	TenSpades   = &Card{TheRank: Ten, TheSuit: Spades}
	NineSpades  = &Card{TheRank: Nine, TheSuit: Spades}
	EightSpades = &Card{TheRank: Eight, TheSuit: Spades}
	SevenSpades = &Card{TheRank: Seven, TheSuit: Spades}
	SixSpades   = &Card{TheRank: Six, TheSuit: Spades}
	FiveSpades  = &Card{TheRank: Five, TheSuit: Spades}
	FourSpades  = &Card{TheRank: Four, TheSuit: Spades}
	ThreeSpades = &Card{TheRank: Three, TheSuit: Spades}
	TwoSpades   = &Card{TheRank: Two, TheSuit: Spades}

	AceHearts   = &Card{TheRank: Ace, TheSuit: Hearts}
	KingHearts  = &Card{TheRank: King, TheSuit: Hearts}
	QueenHearts = &Card{TheRank: Queen, TheSuit: Hearts}
	JackHearts  = &Card{TheRank: Jack, TheSuit: Hearts}
	TenHearts   = &Card{TheRank: Ten, TheSuit: Hearts}
	NineHearts  = &Card{TheRank: Nine, TheSuit: Hearts}
	EightHearts = &Card{TheRank: Eight, TheSuit: Hearts}
	SevenHearts = &Card{TheRank: Seven, TheSuit: Hearts}
	SixHearts   = &Card{TheRank: Six, TheSuit: Hearts}
	FiveHearts  = &Card{TheRank: Five, TheSuit: Hearts}
	FourHearts  = &Card{TheRank: Four, TheSuit: Hearts}
	ThreeHearts = &Card{TheRank: Three, TheSuit: Hearts}
	TwoHearts   = &Card{TheRank: Two, TheSuit: Hearts}

	AceDiamonds   = &Card{TheRank: Ace, TheSuit: Diamonds}
	KingDiamonds  = &Card{TheRank: King, TheSuit: Diamonds}
	QueenDiamonds = &Card{TheRank: Queen, TheSuit: Diamonds}
	JackDiamonds  = &Card{TheRank: Jack, TheSuit: Diamonds}
	TenDiamonds   = &Card{TheRank: Ten, TheSuit: Diamonds}
	NineDiamonds  = &Card{TheRank: Nine, TheSuit: Diamonds}
	EightDiamonds = &Card{TheRank: Eight, TheSuit: Diamonds}
	SevenDiamonds = &Card{TheRank: Seven, TheSuit: Diamonds}
	SixDiamonds   = &Card{TheRank: Six, TheSuit: Diamonds}
	FiveDiamonds  = &Card{TheRank: Five, TheSuit: Diamonds}
	FourDiamonds  = &Card{TheRank: Four, TheSuit: Diamonds}
	ThreeDiamonds = &Card{TheRank: Three, TheSuit: Diamonds}
	TwoDiamonds   = &Card{TheRank: Two, TheSuit: Diamonds}

	AceClubs   = &Card{TheRank: Ace, TheSuit: Clubs}
	KingClubs  = &Card{TheRank: King, TheSuit: Clubs}
	QueenClubs = &Card{TheRank: Queen, TheSuit: Clubs}
	JackClubs  = &Card{TheRank: Jack, TheSuit: Clubs}
	TenClubs   = &Card{TheRank: Ten, TheSuit: Clubs}
	NineClubs  = &Card{TheRank: Nine, TheSuit: Clubs}
	EightClubs = &Card{TheRank: Eight, TheSuit: Clubs}
	SevenClubs = &Card{TheRank: Seven, TheSuit: Clubs}
	SixClubs   = &Card{TheRank: Six, TheSuit: Clubs}
	FiveClubs  = &Card{TheRank: Five, TheSuit: Clubs}
	FourClubs  = &Card{TheRank: Four, TheSuit: Clubs}
	ThreeClubs = &Card{TheRank: Three, TheSuit: Clubs}
	TwoClubs   = &Card{TheRank: Two, TheSuit: Clubs}
)

// Cards returns all 52 unshuffled cards
func Cards() []*Card {
	return []*Card{
		AceSpades, KingSpades, QueenSpades, JackSpades, TenSpades,
		NineSpades, EightSpades, SevenSpades, SixSpades, FiveSpades,
		FourSpades, ThreeSpades, TwoSpades,

		AceHearts, KingHearts, QueenHearts, JackHearts, TenHearts,
		NineHearts, EightHearts, SevenHearts, SixHearts, FiveHearts,
		FourHearts, ThreeHearts, TwoHearts,

		AceDiamonds, KingDiamonds, QueenDiamonds, JackDiamonds, TenDiamonds,
		NineDiamonds, EightDiamonds, SevenDiamonds, SixDiamonds, FiveDiamonds,
		FourDiamonds, ThreeDiamonds, TwoDiamonds,

		AceClubs, KingClubs, QueenClubs, JackClubs, TenClubs,
		NineClubs, EightClubs, SevenClubs, SixClubs, FiveClubs,
		FourClubs, ThreeClubs, TwoClubs,
	}
}

// Cards returns all 52 unshuffled cards order by rank
func CardsOrderByRank() []*Card {
	return []*Card{
		AceSpades, AceHearts, AceClubs, AceDiamonds,
		KingSpades, KingHearts, KingClubs, KingDiamonds,
		QueenSpades, QueenHearts, QueenClubs, QueenDiamonds,
		JackSpades, JackHearts, JackClubs, JackDiamonds,
		TenSpades, TenHearts, TenClubs, TenDiamonds,
		NineSpades, NineHearts, NineClubs, NineDiamonds,
		EightSpades, EightHearts, EightClubs,EightDiamonds,
		SevenSpades, SevenHearts, SevenClubs, SevenDiamonds,
		SixSpades, SixHearts, SixClubs, SixDiamonds,
		FiveSpades, FiveHearts,FiveClubs, FiveDiamonds,
		FourSpades, FourHearts, FourClubs, FourDiamonds,
		ThreeSpades, ThreeHearts,ThreeClubs,ThreeDiamonds,
		TwoSpades, TwoHearts, TwoClubs, TwoDiamonds,
	}
}

type byAceHigh []*Card

func (a byAceHigh) Len() int { return len(a) }

func (a byAceHigh) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a byAceHigh) Less(i, j int) bool {
	iCard, jCard := a[i], a[j]
	iIndex, jIndex := iCard.Rank().indexOf(), jCard.Rank().indexOf()
	return iIndex < jIndex
}

type byAceLow []*Card

func (a byAceLow) Len() int { return len(a) }

func (a byAceLow) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a byAceLow) Less(i, j int) bool {
	iCard, jCard := a[i], a[j]
	iIndex, jIndex := iCard.Rank().aceLowIndexOf(), jCard.Rank().aceLowIndexOf()
	return iIndex < jIndex
}

func allRanks() []Rank {
	return []Rank{Two, Three, Four, Five, Six, Seven, Eight,
		Nine, Ten, Jack, Queen, King, Ace}
}

func allAceLowRanks() []Rank {
	return []Rank{Ace, Two, Three, Four, Five, Six, Seven, Eight,
		Nine, Ten, Jack, Queen, King}
}

func allSuits() []Suit {
	return []Suit{Spades, Hearts, Diamonds, Clubs}
}

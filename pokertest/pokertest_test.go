package pokertest_test

import (
	"testing"

	"github.com/rolends1986/poker/hand"
	"github.com/rolends1986/poker/pokertest"
)

func TestDeck(t *testing.T) {
	cards := pokertest.Cards("Qh", "Ks", "4s")
	actual := []*hand.Card{hand.QueenHearts, hand.KingSpades, hand.FourSpades}
	deck := pokertest.Dealer(cards).Deck()

	for i := 0; i < len(actual); i++ {
		card := deck.Pop()
		if actual[i] != card {
			t.Fatalf("Pop() = %s; want %s; i = %d", card, actual[i], i)
		}
	}
}

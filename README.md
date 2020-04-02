jPker
========

The goal of Poker is to be an open source and fully functioning poker backend in go.  Poker will attempt to support roughly the same feature set as Full Tilt Poker.  

## cmd

cmd holds the executable portions of Poker.  This currently includes a comand line client demoing table functionality.

## hand

The hand package is responsible for poker hand evaluation.  hand is also home to card and deck implementations.  

## pokertest

The pokertest package provides convience methods for testing in the other packages.  For example, pokertest's Dealer produces a Deck with a prearranged series of cards instead of ones in random order.  

## pot

The pot package tracks contributions from players and awards players with winning hands.  It supports hi/lo split pots.  (pot might eventually get merged into table)

## table

The table package provides a table engine to run a poker table.  Turn managment, player action requests, dealing, forced bets, etc are in this package.  An example of a working table is available in th cmd section.  
## util

util is a place for code shared by multiple packages, but otherwise wouldn't be exported.  Might be converted to internal package in go 1.5.

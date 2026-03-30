package model

type Queue struct {
	Current  *Song
	Upcoming []Song
	History  []Song
}

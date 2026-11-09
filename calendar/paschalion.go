package calendar

import (
	"fmt"
	"time"
)

// Pascha returns the date of Pascha for the given year according to the
// Alexandrian (Julian) paschalion, as a Julian date.
//
//	a = Y mod 4, b = Y mod 7, c = Y mod 19
//	d = (19c + 15) mod 30, e = (2a + 4b − d + 34) mod 7
//	n = d + e + 114; month = n div 31, day = n mod 31 + 1
func Pascha(year int) JDate {
	a := mod(year, 4)
	b := mod(year, 7)
	c := mod(year, 19)
	d := (19*c + 15) % 30
	e := (2*a + 4*b - d + 34) % 7
	n := d + e + 114
	return JDate{jdn: julianToJDN(year, n/31, n%31+1)}
}

// KeyOfBoundaries returns the "key of boundaries" (ключ границ) of the year:
// 1 when Pascha falls on 22 March (O.S.) through 35 when it falls on 25 April.
// Everything in the movable cycle of a year is determined by its key (and,
// for the end of the cycle, by the key of the following year).
func KeyOfBoundaries(year int) int {
	p := Pascha(year)
	return p.Sub(MustJDate(year, time.March, 21))
}

// keyLetters are the Church Slavonic letters naming the 35 keys.
var keyLetters = [35]string{
	"А", "Б", "В", "Г", "Д", "Е", "Ж", "Ѕ", "З", "И", "І", "К", "Л", "М", "Н", "О", "П", "Р",
	"С", "Т", "Ꙋ", "Ф", "Х", "Ѿ", "Ц", "Ч", "Ш", "Щ", "Ъ", "Ы", "Ь", "Ѣ", "Ю", "Ѫ", "Ѧ",
}

// KeyLetter returns the Church Slavonic letter of a key of boundaries (1..35).
func KeyLetter(key int) (string, error) {
	if key < 1 || key > 35 {
		return "", fmt.Errorf("calendar: key of boundaries %d out of range 1..35", key)
	}
	return keyLetters[key-1], nil
}

package games

import (
	_ "embed"
	"strings"
)

//go:embed content.txt
var content string

var Challenges []Text

func init() {
	section := ""
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = line
			continue
		}
		parts := strings.Split(line, "|")
		switch section {
		case "[trivia]", "[riddles]":
			if len(parts) != 10 {
				panic("invalid game question bank")
			}
			q := Question{Title: Text{parts[0], parts[1]}}
			for i := 0; i < 4; i++ {
				q.Options[i] = Text{parts[2+i*2], parts[3+i*2]}
			}
			if section == "[trivia]" {
				Trivia = append(Trivia, q)
			} else {
				Riddles = append(Riddles, q)
			}
		case "[kat]", "[challenges]":
			if len(parts) != 2 {
				panic("invalid game prompt bank")
			}
			t := Text{parts[0], parts[1]}
			if section == "[kat]" {
				KatPrompts = append(KatPrompts, t)
			} else {
				Challenges = append(Challenges, t)
			}
		default:
			panic("unknown game content section")
		}
	}
}

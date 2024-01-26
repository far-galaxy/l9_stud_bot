package ssauparser

import (
	"fmt"
	"testing"

	"stud.l9labs.ru/bot/modules/database"
)

func TestLoadJSON(t *testing.T) {
	q := Query{
		YearID:  9,
		Week:    17,
		GroupID: 530996168,
	}

	d, err := LoadJSON(q)
	if err != nil {
		t.Fatal(err)
	}
	var lessons []database.Lesson
	for _, i := range d.Lessons {
		if len(i.Teachers) > 1 {
			fmt.Println("two!")
		}
		if i.Type.ID >= 6 {
			fmt.Println("session!")
		}
		l := database.Lesson{
			NumInShedule: i.Time.NumInSchedule - 1,
			Type:         types[i.Type.ID-1],
			Name:         i.Discipline.Name,
			Comment:      i.Comment,
		}
		for _, t := range i.Teachers {
			l.TeacherId = int64(t.ID)
			for _, g := range i.Groups {
				if g.ID != q.GroupID {
					continue
				}

				l.GroupId = g.ID
				l.SubGroup = g.Subgroup
				for _, w := range i.Weeks {
					if w.Week != int(q.Week) {
						continue
					}
					l.Place = fmt.Sprintf("%s-%s", w.Room.Name, w.Building.Name)
					lessons = append(lessons, l)
				}
			}
		}
	}
	fmt.Print(lessons)
}

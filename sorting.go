package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

const (
	numSessions    = 4
	numArtSessions = 2
	numSciSessions = 2
	random         = true

	artWorkshop = iota
	sciWorkshop
)

func main() {
	groups, err := readGroups("groups.csv")
	if err != nil {
		log.Fatalf("Couldn't read groups: %v", err)
	}

	artWorkshops, err := readWorkshop("artworkshops.csv", "art")
	if err != nil {
		log.Fatalf("Couldn't read art workshop: %v", err)
	}
	//printWorkshops(artWorkshops)

	sciWorkshops, err := readWorkshop("scienceworkshops.csv", "science")
	if err != nil {
		log.Fatalf("Couldn't read science workshop: %v", err)
	}
	//printWorkshops(sciWorkshops)

	log.Printf("====Booking Parent Classes===\n")
	for _, group := range groups {
		for _, parentID := range group.parentIDs {
			if parentID == "0" || parentID == "" {
				continue
			}
			log.Println(parentID)

			kind := idToKind(parentID)
			var workshop *workshop
			var ok bool
			if kind == artWorkshop {
				workshop, ok = artWorkshops[parentID]
				if !ok {
					log.Printf("ID %s not found teacher=%s group=%s\n", parentID, group.teacher, group.name)
				}
			} else {
				workshop, ok = sciWorkshops[parentID]
				if !ok {
					log.Printf("ID %s not found teacher=%s group=%s\n", parentID, group.teacher, group.name)
				}
			}
			booked := bookWorkshopIfAvailable(workshop, group)
			if !booked {
				log.Printf("Unable to book parent ID=%s. teacher=%s group=%s\n", parentID, group.teacher, group.name)
			}
		}
	}

	log.Printf("\n====Booking Art Classes===\n")
	var needsRandomArt []group
	for _, group := range groups {
		sessionsToBook := numArtSessions - group.sessionsBooked(artWorkshop)
		for _, id := range group.artIDs {
			workshop, ok := artWorkshops[id]
			if !ok {
				log.Printf("ID %s not found teacher=%s group=%s\n", id, group.teacher, group.name)
				continue
			}
			booked := bookWorkshopIfAvailable(workshop, group)
			if booked {
				sessionsToBook--
				if sessionsToBook == 0 {
					break
				}
			}
		}

		// Select random session
		for i := 0; i < sessionsToBook; i++ {
			needsRandomArt = append(needsRandomArt, group)
		}
	}

	log.Printf("\n\n====Booking Science Classes===\n")
	var needsRandomSci []group
	for _, group := range groups {
		sessionsToBook := numSciSessions - group.sessionsBooked(sciWorkshop)
		for _, id := range group.sciIDs {
			workshop, ok := sciWorkshops[id]
			if !ok {
				log.Printf("ID %s not found teacher=%s group=%s\n", id, group.teacher, group.name)
				continue
			}
			booked := bookWorkshopIfAvailable(workshop, group)
			if booked {
				sessionsToBook--
				if sessionsToBook == 0 {
					break
				}
			}
		}

		// Select random session
		for i := 0; i < sessionsToBook; i++ {
			needsRandomSci = append(needsRandomSci, group)
		}
	}

	// Assign random sessions if needed
	log.Printf("\n\n====Booking Random Art Classes===\n")
	for _, group := range needsRandomArt {
		log.Printf("NEEDS Random Art %s %s\n", group.teacher, group.name)
		found := false
		for _, workshop := range artWorkshops {
			if !workshop.withinGradeRange(group.grade) {
				continue
			}
			if group.isEnrolledInWorkshop(workshop.id) {
				continue
			}
			sessions := workshop.getAvailableSessions(group)
			for _, session := range sessions {
				if group.workshops[session] == nil {
					workshop.takeSession(session, len(group.students))
					group.workshops[session] = workshop
					found = true
					break

				}
			}
			if found {
				break
			}
		}
		if !found {
			log.Printf("Still not found Art for %s %s\n", group.teacher, group.name)
		}
	}

	log.Printf("\n\n====Booking Random Science Classes===\n")
	for _, group := range needsRandomSci {
		log.Printf("NEEDS Random Sci %s\n", group.name)
		found := false
		for _, workshop := range sciWorkshops {
			if !workshop.withinGradeRange(group.grade) {
				continue
			}
			if group.isEnrolledInWorkshop(workshop.id) {
				continue
			}
			sessions := workshop.getAvailableSessions(group)
			for _, session := range sessions {
				if group.workshops[session] == nil {
					workshop.takeSession(session, len(group.students))
					group.workshops[session] = workshop
					found = true
					break

				}
			}
			if found {
				break
			}
		}
		if !found {
			log.Printf("Still not found Sci for %s %s\n", group.teacher, group.name)
		}
	}

	printGroups(groups)
}

type group struct {
	teacher   string
	grade     int
	name      string
	students  []string
	artIDs    []string
	sciIDs    []string
	parentIDs []string

	workshops []*workshop
}

func (g group) isEnrolledInWorkshop(id string) bool {
	for _, workshop := range g.workshops {
		if workshop == nil {
			continue
		}
		if workshop.id == id {
			return true
		}
	}
	return false
}

func (g group) sessionsBooked(kind int) int {
	booked := 0
	for _, workshop := range g.workshops {
		if workshop != nil {
			workshopKind := idToKind(workshop.id)
			if workshopKind == kind {
				booked++
			}
		}
	}

	return booked
}

func readGroups(file string) ([]group, error) {
	var groups []group

	reader, err := readAndParseCSV(file)
	if err != nil {
		return nil, err
	}

	for {
		// Read each record from csv
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		grade, err := getGrade(record[2])
		if err != nil {
			return nil, err
		}

		artIDs := getRankings(record[5:9], artWorkshop)
		sciIDs := getRankings(record[9:13], sciWorkshop)

		groups = append(groups, group{
			teacher:   record[0],
			grade:     grade,
			name:      record[3],
			students:  strings.Split(record[4], ","),
			artIDs:    artIDs,
			sciIDs:    sciIDs,
			workshops: make([]*workshop, 4),
			parentIDs: strings.Split(record[13], " "),
		})
	}

	return groups, nil
}

func printGroups(groups []group) {
	for _, group := range groups {
		fmt.Printf("Teacher = %s  \n", group.teacher)
		fmt.Printf("Grade = %d  \n", group.grade)
		fmt.Printf("ID = %s-%d-%s  \n", strings.ReplaceAll(group.teacher, " ", "_"), group.grade, strings.ReplaceAll(group.name, " ", "_"))
		fmt.Printf("Students =  %v  \n", group.students)
		/*fmt.Printf("Art Rankings:")
		for _, ranking := range group.artIDs {
			fmt.Printf(" %s", ranking)
		}
		fmt.Println("")
		fmt.Printf("Science Rankings:")
		for _, ranking := range group.sciIDs {
			fmt.Printf(" %s", ranking)
		}*/
		fmt.Println("")
		fmt.Println("Schedule")
		fmt.Println("| ID | Class | Room |")
		fmt.Println("| -- | ----- | ---- |")
		for _, workshop := range group.workshops {
			if workshop != nil {
				fmt.Printf("| %s | %s | %s |\n", workshop.id, workshop.name, workshop.room)
			}
		}
		fmt.Println("\n---\n")
	}
}

type workshop struct {
	kind              string
	id                string
	name              string
	minGrade          int
	maxGrade          int
	sessionCapacities []int
	room              string
}

func (w workshop) getAvailableSessions(group group) []int {
	var availableSessions []int
	numStudents := len(group.students)
	for i, sessionCapacity := range w.sessionCapacities {
		if sessionCapacity >= numStudents {
			if group.workshops[i] == nil {
				availableSessions = append(availableSessions, i)
			}
		}
	}

	return availableSessions
}

func (w workshop) withinGradeRange(grade int) bool {
	if w.minGrade <= grade && grade <= w.maxGrade {
		return true
	}

	return false
}

func (w workshop) takeSession(session, numStudents int) {
	w.sessionCapacities[session] -= numStudents
}

func readWorkshop(file string, kind string) (map[string]*workshop, error) {
	workshops := make(map[string]*workshop)

	reader, err := readAndParseCSV(file)
	if err != nil {
		return nil, err
	}

	for {
		// Read each record from csv
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Parse fields as needed
		id, name, found := strings.Cut(record[0], "-")
		if !found {
			return nil, fmt.Errorf("Invalid workshop name \"%s\"", record[0])
		}
		id = strings.Trim(id, " ")
		name = strings.Trim(name, " ")

		grades := strings.Split(record[1], "-")
		minGrade, err := getGrade(grades[0])
		if err != nil {
			return nil, err
		}
		maxGrade, err := getGrade(grades[1])
		if err != nil {
			return nil, err
		}

		capacity, err := strconv.Atoi(record[6])
		if err != nil {
			return nil, err
		}

		sessionCapacities := make([]int, numSessions)
		for i := 2; i < 6; i++ {
			if strings.ToLower(record[i]) == "y" {
				sessionCapacities[i-2] = capacity
			}
		}

		// Append to array
		workshops[id] = &workshop{
			kind:              kind,
			id:                id,
			name:              name,
			minGrade:          minGrade,
			maxGrade:          maxGrade,
			sessionCapacities: sessionCapacities,
			room:              record[7],
		}
	}

	return workshops, nil
}

func printWorkshops(workshops map[string]*workshop) {
	for id, workshop := range workshops {
		fmt.Printf("Kind: %s\n", workshop.kind)
		fmt.Printf("ID: %s\n", id)
		fmt.Printf("Name: %s\n", workshop.name)
		fmt.Printf("Grade Range: %d-%d\n", workshop.minGrade, workshop.maxGrade)
		fmt.Printf("Capacities:")
		for i := 0; i < 4; i++ {
			fmt.Printf(" %d", workshop.sessionCapacities[i])
		}
		fmt.Println("")
		fmt.Println("-------------")
	}
}

func readAndParseCSV(file string) (*csv.Reader, error) {
	csvFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	//defer csvFile.Close()

	// Parse the file
	reader := csv.NewReader(csvFile)

	// Dump the header line
	_, err = reader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("Empty csv file: %v", err)
	}
	if err != nil {
		log.Fatal(err)
	}

	return reader, nil
}

func getGrade(grade string) (int, error) {
	if strings.ToLower(grade) == "k" {
		return 0, nil
	}

	return strconv.Atoi(grade)
}

func getRankings(rankings []string, kind int) []string {
	var ids []string
	for _, ranking := range rankings {
		if kind == artWorkshop {
			ids = append(ids, strings.Trim("A"+ranking, " "))

		} else {
			ids = append(ids, strings.Trim("S"+ranking, " "))
		}
	}

	return ids
}

func bookWorkshopIfAvailable(workshop *workshop, group group) bool {
	if !workshop.withinGradeRange(group.grade) {
		log.Printf("Mismatched grade id=%s teacher=%s group=%s\n", workshop.id, group.teacher, group.name)
		return false
	}
	if group.isEnrolledInWorkshop(workshop.id) {
		log.Printf("Duplicate workshop id=%s teacher=%s group=%s\n", workshop.id, group.teacher, group.name)
		return false
	}
	sessions := workshop.getAvailableSessions(group)
	if len(sessions) > 0 {
		randSession := sessions[rand.Intn(len(sessions))]
		workshop.takeSession(randSession, len(group.students))
		group.workshops[randSession] = workshop

		return true
	}

	log.Printf("Unable to book session, its full. workshop id=%s teacher=%s group=%s\n", workshop.id, group.teacher, group.name)

	return false
}

func idToKind(id string) int {
	if id[0] == 'A' {
		return artWorkshop
	}

	return sciWorkshop
}

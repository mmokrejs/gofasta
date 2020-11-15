package closest

import (
	"errors"
	"fmt"
	"github.com/cov-ert/gofasta/pkg/encoding"
	"github.com/cov-ert/gofasta/pkg/fastaio"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// getDifferenceMatrix returns a Q x T array with one value per query - target
// comparison. For each query sequence, the lowest cell(s) corresponds to the
// closest target sequence(s)
func getDifferenceMatrix(queryA [][]uint8, targetA [][]uint8) [][]float64 {

	D := make([][]float64, len(queryA))

	for i := 0; i < len(queryA); i++ {
		D[i] = make([]float64, len(targetA))
	}

	alignmentlength := len(queryA[0])

	for queryIndex := 0; queryIndex < len(queryA); queryIndex++ {
		for targetIndex := 0; targetIndex < len(targetA); targetIndex++ {
			differences := 0.0
			denominator := 0.0
			for r := 0; r < alignmentlength; r++ {
				x := targetA[targetIndex][r]
				y := queryA[queryIndex][r]
				different := (x & y) < 16
				same := (x&8 == 8) && x == y

				if different {
					differences += 1.0
				}

				if different || same {
					denominator += 1.0
				}
			}

			D[queryIndex][targetIndex] = (differences / denominator)
		}
	}

	return D
}

// scoreAlignment returns a 1-D array of integers, with higher scores
// for genomes that are more complete (fewer ambiguities)
func scoreAlignment(A [][]uint8) []int {
	S := make([]int, len(A))

	scoreDict := encoding.MakeScoreDict()

	for i := 0; i < len(A); i++ {
		score := 0
		for _, nuc := range A[i] {
			score += scoreDict[nuc]
		}
		S[i] = score
	}
	return S
}

// getMinFloatIndices returns the indices of the minimum values(s) from
// a 1-D array of floats. If there are ties for the lowest value, all
// indices of that value are returned
func getMinFloatIndices(V []float64) []int {

	var min float64
	I := make([]int, 0)

	for i := 0; i < len(V); i++ {

		score := V[i]

		if i == 0 {
			min = score
			I = append(I, i)

		} else if score == min {
			I = append(I, i)

		} else if score < min {
			min = score
			I = []int{i}
		}
	}

	return I
}

// getMaxIntIndices returns the indices of the maximum values(s) from
// a 1-D array of ints. If there are ties for the highest value, all
// indices of that value are returned
func getMaxIntIndices(V []int) []int {

	var max int
	I := make([]int, 0)

	for i := 0; i < len(V); i++ {

		score := V[i]

		if i == 0 {
			max = score
			I = append(I, i)

		} else if score == max {
			I = append(I, i)

		} else if score > max {
			max = score
			I = []int{i}
		}
	}

	return I
}

// getBestTargetIndex is used to select the target sequence that is closest
// to one query sequence. This is based on genetic distance, and ties are
// broken using genome completeness (of the target sequences)
func getBestTargetIndex(differencesV []float64, completenessV []int) int {
	distanceMinIndices := getMinFloatIndices(differencesV)

	var indx int

	if len(distanceMinIndices) > 1 {

		completenesses := make([]int, 0)

		for _, i := range distanceMinIndices {
			completenesses = append(completenesses, completenessV[i])
		}

		completenessMaxIndices := getMaxIntIndices(completenesses)

		indx = distanceMinIndices[completenessMaxIndices[0]]

	} else {
		indx = distanceMinIndices[0]
	}

	return indx
}

// getSNPs returns an array of SNPs between two sequences
func getSNPs(queryV []uint8, targetV []uint8) []string {
	nucDict := encoding.MakeNucDict()
	SNPs := make([]string, 0)

	for r := 0; r < len(targetV); r++ {
		x := queryV[r]
		y := targetV[r]
		different := (x & y) < 16
		if different {
			nucQ := nucDict[x]
			nucT := nucDict[y]
			snp := strconv.Itoa(r+1) + nucQ + nucT
			SNPs = append(SNPs, snp)
		}
	}

	return SNPs
}

// csvRows is a simple struct used for passing the results of processChunk down
// a channel between processes
type csvRows struct {
	id   int
	rows []string
}

// processChunk returns the results from a set of query sequences or sequence.
func processChunk(ch chan csvRows, id int, QA [][]uint8, Qnames []string, TA [][]uint8, Tnames []string, targetScores []int) {

	S := make([]string, len(QA))

	D := getDifferenceMatrix(QA, TA)

	for queryi := 0; queryi < len(QA); queryi++ {
		bestIndx := getBestTargetIndex(D[queryi], targetScores)
		// distance := D[queryi][bestIndx]
		SNPs := getSNPs(QA[queryi], TA[bestIndx])
		SNPdistance := len(SNPs)
		Qname := Qnames[queryi]
		Tname := Tnames[bestIndx]

		row := Qname + "," + Tname + "," + strconv.Itoa(SNPdistance) + "," + strings.Join(SNPs, ";") + "\n"

		S[queryi] = row
	}

	ch <- csvRows{id: id, rows: S}
}

// writeResults writes the csv output file. It's called when all chunks
// have been processed
func writeResults(A [][]string, filepath string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err2 := f.WriteString("query,closest,SNPdistance,SNPs\n")
	if err2 != nil {
		return err2
	}

	for _, chunk := range A {
		for _, row := range chunk {
			_, err3 := f.WriteString(row)
			if err3 != nil {
				return err3
			}
		}
	}

	return nil
}

// Closest finds the closest sequence to each target sequence in a set of
// query sequences. It breaks ties by genome completeness.
func Closest(query string, target string, outfile string, threads int) error {

	QA, Qnames, err := fastaio.PopulateByteArrayGetNames(query)
	if err != nil {
		return err
	}

	if len(QA) != len(Qnames) {
		return errors.New("error parsing query alignment")
	}

	fmt.Printf("number of sequences in query alignment: %d\n", len(QA))

	TA, Tnames, err := fastaio.PopulateByteArrayGetNames(target)
	if err != nil {
		return err
	}

	if len(TA) != len(Tnames) {
		return errors.New("error parsing target alignment")
	}

	fmt.Printf("number of sequences in target alignment: %d\n", len(TA))

	if len(QA[0]) != len(TA[0]) {
		return errors.New("query and target alignments are not the same width")
	}

	targetScores := scoreAlignment(TA)

	ch := make(chan csvRows)

	NGoRoutines := threads

	if NGoRoutines > len(QA) {
		NGoRoutines = len(QA)
	}

	runtime.GOMAXPROCS(NGoRoutines)

	chunkSize := int(math.Floor(float64(len(QA)) / float64(NGoRoutines)))

	for i := 0; i < NGoRoutines; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == NGoRoutines-1 {
			end = len(QA)
		}
		go processChunk(ch, i, QA[start:end], Qnames[start:end], TA, Tnames, targetScores)
	}

	sorted := make([][]string, NGoRoutines)

	for i := 0; i < NGoRoutines; i++ {
		output := <-ch
		sorted[output.id] = output.rows
	}

	close(ch)

	writeResults(sorted, outfile)

	return nil
}

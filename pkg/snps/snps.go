package snps

import (
	"io"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/cov-ert/gofasta/pkg/encoding"
	"github.com/cov-ert/gofasta/pkg/fastaio"
)

// snpLine is a struct for one Fasta record's SNPs
type snpLine struct {
	queryname string
	snps      []string
	idx       int
}

// getSNPs gets the SNPs between the reference and each Fasta record at a time
func getSNPs(refSeq []byte, cFR chan fastaio.EncodedFastaRecord, cSNPs chan snpLine, cErr chan error) {

	DA := encoding.MakeDecodingArray()

	for FR := range cFR {
		SL := snpLine{}
		SL.queryname = FR.ID
		SL.idx = FR.Idx
		SNPs := make([]string, 0)
		for i, nuc := range FR.Seq {
			if (refSeq[i] & nuc) < 16 {
				snpLine := DA[refSeq[i]] + strconv.Itoa(i+1) + DA[nuc]
				SNPs = append(SNPs, snpLine)
			}
		}
		SL.snps = SNPs
		cSNPs <- SL
	}

	return
}

// writeOutput writes the output to stdout or a file as it arrives.
// It uses a map to write things in the same order as they are in the input file.
func writeOutput(w io.Writer, cSNPs chan snpLine, cErr chan error, cWriteDone chan bool) {

	outputMap := make(map[int]snpLine)

	counter := 0

	var err error

	_, err = w.Write([]byte("query,SNPs\n"))
	if err != nil {
		cErr <- err
		return
	}

	for snpLine := range cSNPs {
		outputMap[snpLine.idx] = snpLine

		if SL, ok := outputMap[counter]; ok {
			_, err := w.Write([]byte(SL.queryname + "," + strings.Join(SL.snps, "|") + "\n"))
			if err != nil {
				cErr <- err
				return
			}
			delete(outputMap, counter)
			counter++
		} else {
			continue
		}
	}

	for n := 1; n > 0; {
		if len(outputMap) == 0 {
			n--
			break
		}
		SL := outputMap[counter]
		_, err := w.Write([]byte(SL.queryname + "," + strings.Join(SL.snps, "|") + "\n"))
		if err != nil {
			cErr <- err
			return
		}
		delete(outputMap, counter)
		counter++
	}

	cWriteDone <- true
}

// SNPs annotates snps in a fasta-format alignment with respect to a reference sequence
func SNPs(ref, alignment io.Reader, out io.Writer) error {

	cErr := make(chan error)

	cRef := make(chan fastaio.EncodedFastaRecord)
	cRefDone := make(chan bool)

	cFR := make(chan fastaio.EncodedFastaRecord)
	cFRDone := make(chan bool)

	cSNPs := make(chan snpLine, runtime.NumCPU())
	cSNPsDone := make(chan bool)

	cWriteDone := make(chan bool)

	go fastaio.ReadEncodeAlignment(ref, cRef, cErr, cRefDone)

	var refSeq []byte

	for n := 1; n > 0; {
		select {
		case err := <-cErr:
			return err
		case FR := <-cRef:
			refSeq = FR.Seq
		case <-cRefDone:
			close(cRef)
			n--
		}
	}

	go fastaio.ReadEncodeAlignment(alignment, cFR, cErr, cFRDone)

	go writeOutput(out, cSNPs, cErr, cWriteDone)

	var wgSNPs sync.WaitGroup
	wgSNPs.Add(runtime.NumCPU())

	for n := 0; n < runtime.NumCPU(); n++ {
		go func() {
			getSNPs(refSeq, cFR, cSNPs, cErr)
			wgSNPs.Done()
		}()
	}

	go func() {
		wgSNPs.Wait()
		cSNPsDone <- true
	}()

	for n := 1; n > 0; {
		select {
		case err := <-cErr:
			return err
		case <-cFRDone:
			close(cFR)
			n--
		}
	}

	for n := 1; n > 0; {
		select {
		case err := <-cErr:
			return err
		case <-cSNPsDone:
			close(cSNPs)
			n--
		}
	}

	for n := 1; n > 0; {
		select {
		case err := <-cErr:
			return err
		case <-cWriteDone:
			n--
		}
	}

	return nil
}

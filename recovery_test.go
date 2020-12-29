package kate

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestErasureCodeRecoverSimple(t *testing.T) {
	// Create some random data, with padding...
	fs := NewFFTSettings(5)
	data := make([]Big, fs.maxWidth, fs.maxWidth)
	for i := uint64(0); i < fs.maxWidth/2; i++ {
		asBig(&data[i], i)
	}
	for i := fs.maxWidth / 2; i < fs.maxWidth; i++ {
		data[i] = ZERO
	}
	debugBigs("data", data)
	// Get coefficients for polynomial SLOW_INDICES
	coeffs, err := fs.FFT(data, false)
	if err != nil {
		t.Fatal(err)
	}
	debugBigs("coeffs", coeffs)

	// copy over the 2nd half, leave the first half as nils
	subset := make([]*Big, fs.maxWidth, fs.maxWidth)
	half := fs.maxWidth / 2
	for i := half; i < fs.maxWidth; i++ {
		subset[i] = &coeffs[i]
	}

	debugBigPtrs("subset", subset)
	recovered, err := fs.ErasureCodeRecover(subset)
	if err != nil {
		t.Fatal(err)
	}
	debugBigs("recovered", recovered)
	for i := range recovered {
		if got := &recovered[i]; !equalBig(got, &coeffs[i]) {
			t.Errorf("recovery at index %d got %s but expected %s", i, bigStr(got), bigStr(&coeffs[i]))
		}
	}
	// And recover the original data for good measure
	back, err := fs.FFT(recovered, true)
	if err != nil {
		t.Fatal(err)
	}
	debugBigs("back", back)
	for i := uint64(0); i < half; i++ {
		if got := &back[i]; !equalBig(got, &data[i]) {
			t.Errorf("data at index %d got %s but expected %s", i, bigStr(got), bigStr(&data[i]))
		}
	}
	for i := half; i < fs.maxWidth; i++ {
		if got := &back[i]; !equalZero(got) {
			t.Errorf("expected zero padding in index %d", i)
		}
	}
}

func TestErasureCodeRecover(t *testing.T) {
	// Create some random data, with padding...
	fs := NewFFTSettings(7)
	data := make([]Big, fs.maxWidth, fs.maxWidth)
	for i := uint64(0); i < fs.maxWidth/2; i++ {
		asBig(&data[i], i)
	}
	for i := fs.maxWidth / 2; i < fs.maxWidth; i++ {
		data[i] = ZERO
	}
	debugBigs("data", data)
	// Get coefficients for polynomial SLOW_INDICES
	coeffs, err := fs.FFT(data, false)
	if err != nil {
		t.Fatal(err)
	}
	debugBigs("coeffs", coeffs)

	// Util to pick a random subnet of the values
	randomSubset := func(known uint64, rngSeed uint64) []*Big {
		withMissingValues := make([]*Big, fs.maxWidth, fs.maxWidth)
		for i := range coeffs {
			withMissingValues[i] = &coeffs[i]
		}
		rng := rand.New(rand.NewSource(int64(rngSeed)))
		missing := fs.maxWidth - known
		pruned := rng.Perm(int(fs.maxWidth))[:missing]
		for _, i := range pruned {
			withMissingValues[i] = nil
		}
		return withMissingValues
	}

	// Try different amounts of known indices, and try it in multiple random ways
	var lastKnown uint64 = 0
	for knownRatio := 0.7; knownRatio < 1.0; knownRatio += 0.05 {
		known := uint64(float64(fs.maxWidth) * knownRatio)
		if known == lastKnown {
			continue
		}
		lastKnown = known
		for i := 0; i < 3; i++ {
			t.Run(fmt.Sprintf("random_subset_%d_known_%d", i, known), func(t *testing.T) {
				subset := randomSubset(known, uint64(i))

				debugBigPtrs("subset", subset)
				recovered, err := fs.ErasureCodeRecover(subset)
				if err != nil {
					t.Fatal(err)
				}
				debugBigs("recovered", recovered)
				for i := range recovered {
					if got := &recovered[i]; !equalBig(got, &coeffs[i]) {
						t.Errorf("recovery at index %d got %s but expected %s", i, bigStr(got), bigStr(&coeffs[i]))
					}
				}
				// And recover the original data for good measure
				back, err := fs.FFT(recovered, true)
				if err != nil {
					t.Fatal(err)
				}
				debugBigs("back", back)
				half := uint64(len(back)) / 2
				for i := uint64(0); i < half; i++ {
					if got := &back[i]; !equalBig(got, &data[i]) {
						t.Errorf("data at index %d got %s but expected %s", i, bigStr(got), bigStr(&data[i]))
					}
				}
				for i := half; i < fs.maxWidth; i++ {
					if got := &back[i]; !equalZero(got) {
						t.Errorf("expected zero padding in index %d", i)
					}
				}
			})
		}
	}
}

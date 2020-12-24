package kate

import "fmt"

// FK20 Method to compute all proofs
// Toeplitz multiplication via http://www.netlib.org/utk/people/JackDongarra/etemplates/node384.html
// Single proof method

// A Toeplitz matrix is of the form
//
// t_0     t_(-1) t_(-2) ... t_(1-n)
// t_1     t_0    t_(-1) ... t_(2-n)
// t_2     t_1               .
// .              .          .
// .                 .       .
// .                    .    t(-1)
// t_(n-1)   ...       t_1   t_0
//
// The vector [t_0, t_1, ..., t_(n-2), t_(n-1), 0, t_(1-n), t_(2-n), ..., t_(-2), t_(-1)]
// completely determines the Toeplitz matrix and is called the "toeplitz_coefficients" below

// The composition toeplitz_part3(toeplitz_part2(toeplitz_coefficients, toeplitz_part1(x)))
// compute the matrix-vector multiplication T * x
//
// The algorithm here is written under the assumption x = G1 elements, T scalars
//
// For clarity, vectors in "Fourier space" are written with _fft. So for example, the vector
// xext is the extended x vector (padded with zero), and xext_fft is its Fourier transform.

// Performs the first part of the Toeplitz matrix multiplication algorithm, which is a Fourier
// transform of the vector x extended
func (ks *KateSettings) toeplitzPart1(x []G1) []G1 {
	half := ks.width / 2
	if uint64(len(x)) != half {
		panic(fmt.Errorf("expected width %d (half of settings), got %d", half, len(x)))
	}
	// Extend x with zeros (neutral element of G1)
	xExt := make([]G1, ks.width, ks.width)
	for i := uint64(0); i < half; i++ {
		CopyG1(&xExt[i], &x[i])
	}
	for i := half; i < ks.width; i++ {
		CopyG1(&xExt[i], &zeroG1)
	}
	xExtFFT, err := ks.FFTG1(xExt, false)
	if err != nil {
		panic(fmt.Errorf("FFT G1 failed in toeplitz part 1: %v", err))
	}
	return xExtFFT
}

// Performs the second part of the Toeplitz matrix multiplication algorithm
func (ks *KateSettings) ToeplitzPart2(toeplitzCoeffs []Big, xExtFFT []G1) (hExtFFT []G1) {
	if uint64(len(toeplitzCoeffs)) != ks.width {
		panic("expected toeplitz coeffs to match the width")
	}
	if uint64(len(xExtFFT)) != ks.width {
		panic("expected xExtFFT to match width")
	}
	toeplitzCoeffsFFT, err := ks.FFT(toeplitzCoeffs, false)
	if err != nil {
		panic(fmt.Errorf("FFT failed in toeplitz part 2: %v", err))
	}
	hExtFFT = make([]G1, ks.width, ks.width)
	for i := uint64(0); i < ks.width; i++ {
		mulG1(&hExtFFT[i], &xExtFFT[i], &toeplitzCoeffsFFT[i])
	}
	return hExtFFT
}

// Transform back and return the first half of the vector
func (ks *KateSettings) ToeplitzPart3(hExtFFT []G1) []G1 {
	if uint64(len(hExtFFT)) != ks.width {
		panic("expected hExtFFT to match the width")
	}
	out, err := ks.FFTG1(hExtFFT, true)
	if err != nil {
		panic(fmt.Errorf("toeplitz part 3 err: %v", err))
	}
	// Only the top half is the Toeplitz product, the rest is padding
	return out[:ks.width/2]
}

func (ks *KateSettings) toeplitzCoeffsStep(polynomial []Big) []Big {
	n := ks.width / 2
	if uint64(len(polynomial)) != n {
		panic("bad polynomial length")
	}
	// [last poly item] + [0]*(n+1) + [poly items except first and last]
	toeplitzCoeffs := make([]Big, ks.width, ks.width)
	CopyBigNum(&toeplitzCoeffs[0], &polynomial[n-1])
	for i := uint64(0); i < n+1; i++ {
		CopyBigNum(&toeplitzCoeffs[i+1], &ZERO)
	}
	for i := n + 1; i < ks.width; i++ {
		CopyBigNum(&toeplitzCoeffs[i], &polynomial[i-n])
	}
	return toeplitzCoeffs
}

// Compute all n (single) proofs according to FK20 method
func (ks *KateSettings) FK20Single(polynomial []Big) []G1 {
	toeplitzCoeffs := ks.toeplitzCoeffsStep(polynomial)
	// Compute the vector h from the paper using a Toeplitz matrix multiplication
	hExtFFT := ks.ToeplitzPart2(toeplitzCoeffs, ks.xExtFFT)
	h := ks.ToeplitzPart3(hExtFFT)

	out, err := ks.FFTG1(h, false)
	if err != nil {
		panic(err)
	}
	return out
}

// Special version of the FK20 for the situation of data availability checks:
// The upper half of the polynomial coefficients is always 0, so we do not need to extend to twice the size
// for Toeplitz matrix multiplication
func (ks *KateSettings) FK20SingleDAOptimized(polynomial []Big) []G1 {
	if uint64(len(polynomial)) != ks.width {
		panic(fmt.Errorf(
			"expected input of length %d (incl half of zeroes) to match precomputed settings length %d",
			len(polynomial), ks.width))
	}
	n := ks.width / 2
	for i := n; i < ks.width; i++ {
		if !equalZero(&polynomial[i]) {
			panic("bad input, second half should be zeroed")
		}
	}
	reducedPoly := polynomial[:n]
	toeplitzCoeffs := ks.toeplitzCoeffsStep(reducedPoly)
	// Compute the vector h from the paper using a Toeplitz matrix multiplication
	hExtFFT := ks.ToeplitzPart2(toeplitzCoeffs, ks.xExtFFT)
	h := ks.ToeplitzPart3(hExtFFT)

	// Now redo the padding before final step.
	// Instead of copying h into a new extended array, just reuse the old capacity.
	h = h[:ks.width]
	for i := n; i < ks.width; i++ {
		CopyG1(&h[i], &zeroG1)
	}
	out, err := ks.FFTG1(h, false)
	if err != nil {
		panic(err)
	}
	return out
}

// Computes all the KZG proofs for data availability checks. This involves sampling on the double domain
// and reordering according to reverse bit order
func (ks *KateSettings) DAUsingFK20(polynomial []Big) []G1 {
	n := uint64(len(polynomial))
	if n*2 != ks.width {
		panic("expected poly contents half the size of the Kate settings")
	}
	extendedPolynomial := make([]Big, ks.width, ks.width)
	for i := uint64(0); i < n; i++ {
		CopyBigNum(&extendedPolynomial[i], &polynomial[i])
	}
	for i := n; i < ks.width; i++ {
		CopyBigNum(&extendedPolynomial[i], &ZERO)
	}
	allProofs := ks.FK20SingleDAOptimized(extendedPolynomial)
	// change to reverse bit order.
	reverseBitOrderG1(allProofs)
	return allProofs
}

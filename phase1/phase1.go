package phase1

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/bnb-chain/zkbnb-setup/common"
	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"math"
	"math/big"
	"os"
)

func Transform(inputPath, outputPath string, inPower, outPower byte) error {
	// Input file is in uncompressed representation
	const G1Size = 64
	const G2Size = 128

	// Formatted as
	// Hash, 2^(2n)-1[TauG1], 2^n[TauG2], 2^n[AlphaG1], 2^n[BetaG1], BetaG2
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	// Output file is in compressed representation
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Write header
	header := Header{Power: outPower, Contributions: 0}
	if err := header.writeTo(outputFile); err != nil {
		return err
	}

	inN := int(math.Pow(2, float64(inPower)))
	outN := int(math.Pow(2, float64(outPower)))

	var posTauG1 int64 = 64
	var posTauG2 int64 = posTauG1 + int64(2*inN-1)*G1Size
	var posAlphaG1 int64 = posTauG2 + int64(inN)*G2Size
	var posBetaG1 int64 = posAlphaG1 + int64(inN)*G1Size
	var posBetaG2 int64 = posBetaG1 + int64(inN)*G1Size

	// Transform TauG1
	fmt.Println("Transforming TauG1")
	if err := transformG1(inputFile, outputFile, posTauG1, 2*outN-1); err != nil {
		return err
	}

	// Transform AlphaG1
	fmt.Println("Transforming AlphaG1")
	if err := transformG1(inputFile, outputFile, posAlphaG1, outN); err != nil {
		return err
	}

	// Transform BetaG1
	fmt.Println("Transforming BetaG1")
	if err := transformG1(inputFile, outputFile, posBetaG1, outN); err != nil {
		return err
	}

	// Transform TauG2
	fmt.Println("Transforming TauG2")
	if err := transformG2(inputFile, outputFile, posTauG2, outN); err != nil {
		return err
	}

	// Transform BetaG2
	fmt.Println("Transforming BetaG2")
	if err := transformG2(inputFile, outputFile, posBetaG2, 1); err != nil {
		return err
	}

	return nil
}

func Initialize(power byte, outputPath string) error {
	_, _, g1, g2 := bn254.Generators()
	// output outputFile
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	var header Header

	header.Power = power
	N := int(math.Pow(2, float64(power)))
	fmt.Printf("Power %d supports up to %d constraints\n", power, N)

	// Write the header
	header.writeTo(outputFile)

	// Use buffered IO to write parameters efficiently
	buffSize := int(math.Pow(2, 20))
	writer := bufio.NewWriterSize(outputFile, buffSize)
	defer writer.Flush()

	// BN254 encoder using compressed representation of points to save storage space
	enc := bn254.NewEncoder(writer)

	// In the initialization, τ = α = β = 1, so we are writing the generators directly
	// Write [τ⁰]₁, [τ¹]₁, [τ²]₁, …, [τ²ᴺ⁻²]₁
	fmt.Println("1. Writing TauG1")
	for i := 0; i < 2*N-1; i++ {
		if err := enc.Encode(&g1); err != nil {
			return err
		}
	}

	// Write α[τ⁰]₁, α[τ¹]₁, α[τ²]₁, …, α[τᴺ⁻¹]₁
	fmt.Println("2. Writing AlphaTauG1")
	for i := 0; i < N; i++ {
		if err := enc.Encode(&g1); err != nil {
			return err
		}
	}

	// Write β[τ⁰]₁, β[τ¹]₁, β[τ²]₁, …, β[τᴺ⁻¹]₁
	fmt.Println("3. Writing BetaTauG1")
	for i := 0; i < N; i++ {
		if err := enc.Encode(&g1); err != nil {
			return err
		}
	}

	// Write {[τ⁰]₂, [τ¹]₂, [τ²]₂, …, [τᴺ⁻¹]₂}
	fmt.Println("4. Writing TauG2")
	for i := 0; i < N; i++ {
		if err := enc.Encode(&g2); err != nil {
			return err
		}
	}

	// Write [β]₂
	fmt.Println("5. Writing BetaG2")
	enc.Encode(&g2)

	fmt.Println("Initialization has been completed successfully")
	return nil
}

func Contribute(inputPath, outputPath string) error {
	// Input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	// Output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Read/Write header with extra contribution
	var header Header
	if err := header.ReadFrom(inputFile); err != nil {
		return err
	}
	fmt.Printf("Power := %d and  #Contributions := %d\n", header.Power, header.Contributions)
	N := int(math.Pow(2, float64(header.Power)))
	header.Contributions++
	if err := header.writeTo(outputFile); err != nil {
		return err
	}

	// Use buffered IO to write parameters efficiently
	reader := bufio.NewReader(inputFile)
	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	dec := bn254.NewDecoder(reader)
	enc := bn254.NewEncoder(writer)

	// Sample toxic parameters
	fmt.Println("Sampling toxic parameters Tau, Alpha, and Beta")
	var tau, alpha, beta, one fr.Element
	tau.SetRandom()
	alpha.SetRandom()
	beta.SetRandom()
	one.SetOne()

	var contribution Contribution
	var firstG1 *bn254.G1Affine
	var firstG2 *bn254.G2Affine

	// Process Tau section
	fmt.Println("Processing TauG1")
	if firstG1, err = scaleG1(dec, enc, 2*N-1, &tau, nil); err != nil {
		return err
	}
	contribution.G1.Tau.Set(firstG1)

	// Process AlphaTauG1 section
	fmt.Println("Processing AlphaTauG1")
	if firstG1, err = scaleG1(dec, enc, N, &tau, &alpha); err != nil {
		return err
	}
	contribution.G1.Alpha.Set(firstG1)

	// Process BetaTauG1 section
	fmt.Println("Processing BetaTauG1")
	if firstG1, err = scaleG1(dec, enc, N, &tau, &beta); err != nil {
		return err
	}
	contribution.G1.Beta.Set(firstG1)

	// Process TauG2 section
	fmt.Println("Processing TauG2")
	if firstG2, err = scaleG2(dec, enc, N, &tau); err != nil {
		return err
	}
	contribution.G2.Tau.Set(firstG2)

	// Process BetaG2 section
	fmt.Println("Processing BetaG2")
	var betaG2 bn254.G2Affine
	var betaBi big.Int
	if err := dec.Decode(&betaG2); err != nil {
		return err
	}
	beta.BigInt(&betaBi)
	betaG2.ScalarMultiplication(&betaG2, &betaBi)
	if err := enc.Encode(&betaG2); err != nil {
		return err
	}
	contribution.G2.Beta.Set(&betaG2)

	// Copy old contributions
	nExistingContributions := int(header.Contributions - 1)
	var c Contribution
	for i := 0; i < nExistingContributions; i++ {
		if _, err := c.ReadFrom(reader); err != nil {
			return err
		}
		if _, err := c.writeTo(writer); err != nil {
			return err
		}
	}

	// Get hash of previous contribution
	var prevHash []byte
	if nExistingContributions == 0 {
		prevHash = nil
	} else {
		prevHash = c.Hash
	}

	// Generate public keys
	contribution.PublicKeys.Tau = common.GenPublicKey(tau, prevHash, 1)
	contribution.PublicKeys.Alpha = common.GenPublicKey(alpha, prevHash, 2)
	contribution.PublicKeys.Beta = common.GenPublicKey(beta, prevHash, 3)
	contribution.Hash = computeHash(&contribution)

	// Write the contribution
	contribution.writeTo(writer)

	fmt.Println("Contirbution has been successful!")
	fmt.Println("Contribution Hash := ", hex.EncodeToString(contribution.Hash))

	return nil
}

func ContributeServer(inputPath, outputPath string, tau, alpha, beta, one fr.Element) (string, error) {
	// Input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return "", err
	}
	defer inputFile.Close()

	// Output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer outputFile.Close()

	// Read/Write header with extra contribution
	var header Header
	if err := header.ReadFrom(inputFile); err != nil {
		return "", err
	}
	fmt.Printf("Power := %d and  #Contributions := %d\n", header.Power, header.Contributions)
	N := int(math.Pow(2, float64(header.Power)))
	header.Contributions++
	if err := header.writeTo(outputFile); err != nil {
		return "", err
	}

	// Use buffered IO to write parameters efficiently
	reader := bufio.NewReader(inputFile)
	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	dec := bn254.NewDecoder(reader)
	enc := bn254.NewEncoder(writer)

	// Process Tau section
	//fmt.Println("Processing TauG1")
	var firstG1 *bn254.G1Affine
	if firstG1, err = scaleG1(dec, enc, 2*N-1, &tau, nil); err != nil {
		return "", err
	}
	//fmt.Printf("firstG1.X: %s\n", firstG1.X.String())
	//fmt.Printf("firstG1.Y: %s\n", firstG1.Y.String())
	var contribution Contribution
	contribution.G1.Tau.Set(firstG1)

	// Process AlphaTauG1 section
	//fmt.Println("Processing AlphaTauG1")
	if firstG1, err = scaleG1(dec, enc, N, &tau, &alpha); err != nil {
		return "", err
	}
	//fmt.Printf("firstG1.X: %s\n", firstG1.X.String())
	//fmt.Printf("firstG1.Y: %s\n", firstG1.Y.String())
	contribution.G1.Alpha.Set(firstG1)

	// Process BetaTauG1 section
	//fmt.Println("Processing BetaTauG1")
	if firstG1, err = scaleG1(dec, enc, N, &tau, &beta); err != nil {
		return "", err
	}
	//fmt.Printf("firstG1.X: %s\n", firstG1.X.String())
	//fmt.Printf("firstG1.Y: %s\n", firstG1.Y.String())
	contribution.G1.Beta.Set(firstG1)

	// Process TauG2 section
	//fmt.Println("Processing TauG2")
	var firstG2 *bn254.G2Affine
	if firstG2, err = scaleG2(dec, enc, N, &tau); err != nil {
		return "", err
	}
	//fmt.Printf("firstG2.X: %s\n", firstG2.X.String())
	//fmt.Printf("firstG2.Y: %s\n", firstG2.Y.String())
	contribution.G2.Tau.Set(firstG2)

	// Process BetaG2 section
	//fmt.Println("Processing BetaG2")
	var betaG2 bn254.G2Affine
	var betaBi big.Int
	if err := dec.Decode(&betaG2); err != nil {
		return "", err
	}
	beta.BigInt(&betaBi)
	betaG2.ScalarMultiplication(&betaG2, &betaBi)
	//fmt.Printf("betaG2.X: %s\n", betaG2.X.String())
	//fmt.Printf("betaG2.Y: %s\n", betaG2.Y.String())
	if err := enc.Encode(&betaG2); err != nil {
		return "", err
	}
	contribution.G2.Beta.Set(&betaG2)

	// Copy old contributions
	nExistingContributions := int(header.Contributions - 1)
	var c Contribution
	if nExistingContributions == 1 {
		if _, err := c.ReadFrom(reader); err != nil {
			return "", err
		}
		if _, err := c.writeTo(writer); err != nil {
			return "", err
		}
	}
	if nExistingContributions > 1 {
		if _, err := c.ReadFrom(reader); err != nil {
			return "", err
		}
		if _, err := c.ReadFrom(reader); err != nil {
			return "", err
		}
		if _, err := c.writeTo(writer); err != nil {
			return "", err
		}
	}

	// Get hash of previous contribution
	var prevHash []byte
	if nExistingContributions == 0 {
		prevHash = nil
	} else {
		prevHash = c.Hash
	}

	// Generate public keys
	var s fr.Element
	s.SetString("5349450036908408402062633344441795636900283135072291564543572822568951905530")
	fmt.Println("s:", s.String())
	contribution.PublicKeys.Tau = common.GenPublicKey_(tau, prevHash, 1, &s)
	contribution.PublicKeys.Alpha = common.GenPublicKey_(alpha, prevHash, 2, &s)
	contribution.PublicKeys.Beta = common.GenPublicKey_(beta, prevHash, 3, &s)
	contribution.Hash = computeHash(&contribution)

	// Write the contribution
	contribution.writeTo(writer)

	fmt.Println("Contribution has been successful!")
	return hex.EncodeToString(contribution.Hash), nil
}

func ContributeServerAll(inputPath, outputPath string, tau, alpha, beta, one fr.Element) (string, error) {
	// Input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return "", err
	}
	defer inputFile.Close()

	// Output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer outputFile.Close()

	// Read/Write header with extra contribution
	var header Header
	if err := header.ReadFrom(inputFile); err != nil {
		return "", err
	}
	fmt.Printf("Power := %d and  #Contributions := %d\n", header.Power, header.Contributions)
	N := int(math.Pow(2, float64(header.Power)))
	header.Contributions++
	if err := header.writeTo(outputFile); err != nil {
		return "", err
	}

	// Use buffered IO to write parameters efficiently
	reader := bufio.NewReader(inputFile)
	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	dec := bn254.NewDecoder(reader)
	enc := bn254.NewEncoder(writer)

	// Process Tau section
	//fmt.Println("Processing TauG1")
	var firstG1 *bn254.G1Affine
	if firstG1, err = scaleG1(dec, enc, 2*N-1, &tau, nil); err != nil {
		return "", err
	}
	var contribution Contribution
	contribution.G1.Tau.Set(firstG1)

	// Process AlphaTauG1 section
	//fmt.Println("Processing AlphaTauG1")
	if firstG1, err = scaleG1(dec, enc, N, &tau, &alpha); err != nil {
		return "", err
	}
	contribution.G1.Alpha.Set(firstG1)

	// Process BetaTauG1 section
	//fmt.Println("Processing BetaTauG1")
	if firstG1, err = scaleG1(dec, enc, N, &tau, &beta); err != nil {
		return "", err
	}
	contribution.G1.Beta.Set(firstG1)

	// Process TauG2 section
	//fmt.Println("Processing TauG2")
	var firstG2 *bn254.G2Affine
	if firstG2, err = scaleG2(dec, enc, N, &tau); err != nil {
		return "", err
	}
	contribution.G2.Tau.Set(firstG2)

	// Process BetaG2 section
	//fmt.Println("Processing BetaG2")
	var betaG2 bn254.G2Affine
	var betaBi big.Int
	if err := dec.Decode(&betaG2); err != nil {
		return "", err
	}
	beta.BigInt(&betaBi)
	betaG2.ScalarMultiplication(&betaG2, &betaBi)
	if err := enc.Encode(&betaG2); err != nil {
		return "", err
	}
	contribution.G2.Beta.Set(&betaG2)

	// Copy old contributions
	nExistingContributions := int(header.Contributions - 1)
	var c Contribution
	for i := 0; i < nExistingContributions; i++ {
		if _, err := c.ReadFrom(reader); err != nil {
			return "", err
		}
		if _, err := c.writeTo(writer); err != nil {
			return "", err
		}
	}

	// Get hash of previous contribution
	var prevHash []byte
	if nExistingContributions == 0 {
		prevHash = nil
	} else {
		prevHash = c.Hash
	}

	// Generate public keys
	var s fr.Element
	s.SetString("5349450036908408402062633344441795636900283135072291564543572822568951905530")
	fmt.Println("s:", s.String())
	contribution.PublicKeys.Tau = common.GenPublicKey_(tau, prevHash, 1, &s)
	contribution.PublicKeys.Alpha = common.GenPublicKey_(alpha, prevHash, 2, &s)
	contribution.PublicKeys.Beta = common.GenPublicKey_(beta, prevHash, 3, &s)
	contribution.Hash = computeHash(&contribution)

	// Write the contribution
	contribution.writeTo(writer)

	fmt.Println("Contribution has been successful!")
	return hex.EncodeToString(contribution.Hash), nil
}

func Verify(inputPath, transformedPath string) error {
	// Input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	// Read header
	var header Header
	if err := header.ReadFrom(inputFile); err != nil {
		return err
	}
	fmt.Printf("Power := %d and  #Contributions := %d\n", header.Power, header.Contributions)
	N := int(math.Pow(2, float64(header.Power)))

	// Use buffered IO to write parameters efficiently
	buffSize := int(math.Pow(2, 20))
	reader := bufio.NewReaderSize(inputFile, buffSize)
	dec := bn254.NewDecoder(reader)

	fmt.Println("Processing TauG1")
	tau1L1, tau1L2, err := linearCombinationG1(dec, 2*N-1)
	if err != nil {
		return err
	}

	fmt.Println("Processing AlphaTauG1")
	alphaTau1L1, alphaTau1L2, err := linearCombinationG1(dec, N)
	if err != nil {
		return err
	}

	fmt.Println("Processing BetaTauG1")
	betaTau1L1, betaTau1L2, err := linearCombinationG1(dec, N)
	if err != nil {
		return err
	}

	fmt.Println("Processing TauG2")
	tau2L1, tau2L2, err := linearCombinationG2(dec, N)
	if err != nil {
		return err
	}

	fmt.Println("Processing BetaG2")
	var betaG2 bn254.G2Affine
	if err = dec.Decode(&betaG2); err != nil {
		return err
	}

	// Verify contributions
	var current Contribution
	prev, err := defaultContribution(transformedPath)
	if err != nil {
		return err
	}
	for i := 0; i < int(header.Contributions); i++ {
		current.ReadFrom(reader)
		fmt.Printf("Verifying contribution %d with Hash := %s\n", i+1, hex.EncodeToString(current.Hash))
		if err := verifyContribution(current, prev); err != nil {
			return err
		}
		prev = current
	}

	// Verify consistency of parameters update
	_, _, g1, g2 := bn254.Generators()
	// Read and verify TauG1
	fmt.Println("Verifying powers of TauG1")
	if !common.SameRatio(tau1L1, tau1L2, current.G2.Tau, g2) {
		return errors.New("failed pairing check")
	}

	// Read and verify AlphaTauG1
	fmt.Println("Verifying powers of AlphaTauG1")
	if !common.SameRatio(alphaTau1L1, alphaTau1L2, current.G2.Tau, g2) {
		return errors.New("failed pairing check")
	}

	// Read and verify BetaTauG1
	fmt.Println("Verifying powers of BetaTauG1")
	if !common.SameRatio(betaTau1L1, betaTau1L2, current.G2.Tau, g2) {
		return errors.New("failed pairing check")
	}

	// Read and verify TauG2
	fmt.Println("Verifying powers of TauG2")
	if !common.SameRatio(g1, current.G1.Tau, tau2L1, tau2L2) {
		return errors.New("failed pairing check")
	}

	// Verify BetaG2
	fmt.Println("Verifying powers of BetaG2")
	if !betaG2.Equal(&current.G2.Beta) {
		return errors.New("failed verifying update of Beta")
	}

	fmt.Println("Contributions verification has been successful")
	return nil
}

func Verify_(inputPath, transformedPath string) error {
	// Input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	// Read header
	var header Header
	if err := header.ReadFrom(inputFile); err != nil {
		return err
	}
	fmt.Printf("Power := %d and  #Contributions := %d\n", header.Power, header.Contributions)
	N := int(math.Pow(2, float64(header.Power)))

	// Use buffered IO to write parameters efficiently
	buffSize := int(math.Pow(2, 20))
	reader := bufio.NewReaderSize(inputFile, buffSize)
	dec := bn254.NewDecoder(reader)

	//fmt.Println("Processing TauG1")
	tau1L1, tau1L2, err := linearCombinationG1(dec, 2*N-1)
	if err != nil {
		return err
	}

	//fmt.Println("Processing AlphaTauG1")
	alphaTau1L1, alphaTau1L2, err := linearCombinationG1(dec, N)
	if err != nil {
		return err
	}

	//fmt.Println("Processing BetaTauG1")
	betaTau1L1, betaTau1L2, err := linearCombinationG1(dec, N)
	if err != nil {
		return err
	}

	//fmt.Println("Processing TauG2")
	tau2L1, tau2L2, err := linearCombinationG2(dec, N)
	if err != nil {
		return err
	}

	//fmt.Println("Processing BetaG2")
	var betaG2 bn254.G2Affine
	if err = dec.Decode(&betaG2); err != nil {
		return err
	}

	// Verify contributions
	var current Contribution
	var prev Contribution
	if header.Contributions == 1 {
		prev, err = defaultContribution(transformedPath)
		if err != nil {
			return err
		}
	} else {
		prev.ReadFrom(reader)
	}
	current.ReadFrom(reader)
	fmt.Printf("the prev contribution %d with Hash := %s\n", header.Contributions-1, hex.EncodeToString(prev.Hash))
	fmt.Printf("Verifying contribution %d with Hash := %s\n", header.Contributions, hex.EncodeToString(current.Hash))
	if err := verifyContribution(current, prev); err != nil {
		return err
	}

	// Verify consistency of parameters update
	_, _, g1, g2 := bn254.Generators()
	// Read and verify TauG1
	//fmt.Println("Verifying powers of TauG1")
	if !common.SameRatio(tau1L1, tau1L2, current.G2.Tau, g2) {
		return errors.New("failed pairing check")
	}

	// Read and verify AlphaTauG1
	//fmt.Println("Verifying powers of AlphaTauG1")
	if !common.SameRatio(alphaTau1L1, alphaTau1L2, current.G2.Tau, g2) {
		return errors.New("failed pairing check")
	}

	// Read and verify BetaTauG1
	//fmt.Println("Verifying powers of BetaTauG1")
	if !common.SameRatio(betaTau1L1, betaTau1L2, current.G2.Tau, g2) {
		return errors.New("failed pairing check")
	}

	// Read and verify TauG2
	//fmt.Println("Verifying powers of TauG2")
	if !common.SameRatio(g1, current.G1.Tau, tau2L1, tau2L2) {
		return errors.New("failed pairing check")
	}

	// Verify BetaG2
	//fmt.Println("Verifying powers of BetaG2")
	if !betaG2.Equal(&current.G2.Beta) {
		return errors.New("failed verifying update of Beta")
	}

	fmt.Println("Contributions verification has been successful")
	return nil
}

package xapi

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
)

type txIDAnalysis struct {
	Samples            int
	OperationFilter    string
	Operations         int
	MinEncodedLength   int
	MaxEncodedLength   int
	MinDecodedLength   int
	MaxDecodedLength   int
	AverageDecodedLen  float64
	AverageBitDelta    float64
	SameSecondPairs    int
	SameSecondAvgDelta float64
	SameMSPairs        int
	SameMSAvgDelta     float64
	CT0Variants        int
	AuthVariants       int
	StableBits         int
	BiasedBits         int
	BitProbabilities   []float64
	StablePrefixBytes  int
	StableByteCount    int
	FirstBytes         []model.DoctorCheck
	HotBytes           []model.DoctorCheck
	TopOperations      []model.DoctorCheck
	SaltPatterns       []model.DoctorCheck
	UniqueSalts        int
	SameSaltOps        int
}

func AnalyzeTXIDTrace(filePath string, operationFilter string) (txIDAnalysis, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return txIDAnalysis{}, err
	}
	defer file.Close()

	analysis := txIDAnalysis{MinEncodedLength: math.MaxInt, MinDecodedLength: math.MaxInt, OperationFilter: strings.TrimSpace(operationFilter)}
	counts := map[string]int{}
	prevByOp := map[string][]byte{}
	ct0Set := map[string]bool{}
	authSet := map[string]bool{}
	bucketPrev := map[string][]byte{}
	msBucketPrev := map[string][]byte{}
	saltsByOp := map[string]string{}
	saltSet := map[string]bool{}
	var byteSets []map[byte]bool
	var bitOnes []int
	var deltaTotal float64
	var deltaCount int
	var sameSecondDeltaTotal float64
	var sameSecondDeltaCount int
	var sameMSDeltaTotal float64
	var sameMSDeltaCount int
	var decodedTotal int

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec txIDTraceRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		if analysis.OperationFilter != "" && rec.Operation != analysis.OperationFilter {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(padBase64(rec.TxID))
		if err != nil {
			continue
		}
		analysis.Samples++
		counts[rec.Operation]++
		if rec.CT0Hash != "" {
			ct0Set[rec.CT0Hash] = true
		}
		if rec.AuthorizationHash != "" {
			authSet[rec.AuthorizationHash] = true
		}

		// Track salt patterns
		salt := rec.TxIDSalt
		if salt == "" && len(decoded) >= 70 {
			salt = fmt.Sprintf("%x", decoded[41:70])
		}
		if salt != "" {
			saltSet[salt] = true
			if existing, ok := saltsByOp[rec.Operation]; ok && existing == salt {
				analysis.SameSaltOps++
			}
			saltsByOp[rec.Operation] = salt
		}

		encodedLen := len(rec.TxID)
		decodedLen := len(decoded)
		if len(byteSets) < decodedLen {
			for i := len(byteSets); i < decodedLen; i++ {
				byteSets = append(byteSets, map[byte]bool{})
			}
		}
		if len(bitOnes) < decodedLen*8 {
			bitOnes = append(bitOnes, make([]int, decodedLen*8-len(bitOnes))...)
		}
		for i, b := range decoded {
			byteSets[i][b] = true
			for bit := 0; bit < 8; bit++ {
				if (b & (1 << uint(7-bit))) != 0 {
					bitOnes[i*8+bit]++
				}
			}
		}
		decodedTotal += decodedLen
		if encodedLen < analysis.MinEncodedLength {
			analysis.MinEncodedLength = encodedLen
		}
		if encodedLen > analysis.MaxEncodedLength {
			analysis.MaxEncodedLength = encodedLen
		}
		if decodedLen < analysis.MinDecodedLength {
			analysis.MinDecodedLength = decodedLen
		}
		if decodedLen > analysis.MaxDecodedLength {
			analysis.MaxDecodedLength = decodedLen
		}
		if prev, ok := prevByOp[rec.Operation]; ok {
			deltaTotal += float64(bitHammingDistance(prev, decoded))
			deltaCount++
		}
		bucketKey := rec.Operation + "|" + fmt.Sprintf("%d", rec.TimestampMS/1000)
		if prev, ok := bucketPrev[bucketKey]; ok {
			sameSecondDeltaTotal += float64(bitHammingDistance(prev, decoded))
			sameSecondDeltaCount++
		}
		msBucketKey := rec.Operation + "|" + fmt.Sprintf("%d", rec.TimestampMS)
		if prev, ok := msBucketPrev[msBucketKey]; ok {
			sameMSDeltaTotal += float64(bitHammingDistance(prev, decoded))
			sameMSDeltaCount++
		}
		prevByOp[rec.Operation] = decoded
		bucketPrev[bucketKey] = decoded
		msBucketPrev[msBucketKey] = decoded
	}
	if err := scanner.Err(); err != nil {
		return txIDAnalysis{}, err
	}
	if analysis.Samples == 0 {
		return txIDAnalysis{}, fmt.Errorf("no valid txid samples in %s", filePath)
	}
	analysis.Operations = len(counts)
	analysis.CT0Variants = len(ct0Set)
	analysis.AuthVariants = len(authSet)
	analysis.UniqueSalts = len(saltSet)

	// Collect salt patterns for display
	saltIdx := 0
	for salt := range saltSet {
		if saltIdx >= 8 {
			break
		}
		analysis.SaltPatterns = append(analysis.SaltPatterns, model.DoctorCheck{
			Name:    fmt.Sprintf("salt-%d", saltIdx),
			Status:  salt[:16] + "...",
			Details: fmt.Sprintf("%d bytes", len(salt)/2),
		})
		saltIdx++
	}

	analysis.AverageDecodedLen = float64(decodedTotal) / float64(analysis.Samples)
	if deltaCount > 0 {
		analysis.AverageBitDelta = deltaTotal / float64(deltaCount)
	}
	analysis.SameSecondPairs = sameSecondDeltaCount
	if sameSecondDeltaCount > 0 {
		analysis.SameSecondAvgDelta = sameSecondDeltaTotal / float64(sameSecondDeltaCount)
	}
	analysis.SameMSPairs = sameMSDeltaCount
	if sameMSDeltaCount > 0 {
		analysis.SameMSAvgDelta = sameMSDeltaTotal / float64(sameMSDeltaCount)
	}
	analysis.BitProbabilities = make([]float64, len(bitOnes))
	for i, ones := range bitOnes {
		prob := float64(ones) / float64(analysis.Samples)
		analysis.BitProbabilities[i] = prob
		if prob == 0 || prob == 1 {
			analysis.StableBits++
		}
		if prob <= 0.1 || prob >= 0.9 {
			analysis.BiasedBits++
		}
	}

	type byteVariance struct {
		index int
		count int
	}
	var byteStats []byteVariance
	for i, values := range byteSets {
		count := len(values)
		if count == 1 {
			analysis.StableByteCount++
			if analysis.StablePrefixBytes == i {
				analysis.StablePrefixBytes++
			}
		}
		byteStats = append(byteStats, byteVariance{index: i, count: count})
		if i < 8 {
			analysis.FirstBytes = append(analysis.FirstBytes, model.DoctorCheck{
				Name:    fmt.Sprintf("first-byte-%02d", i),
				Status:  fmt.Sprintf("%d", count),
				Details: "unique values",
			})
		}
	}
	sort.Slice(byteStats, func(i, j int) bool {
		if byteStats[i].count == byteStats[j].count {
			return byteStats[i].index < byteStats[j].index
		}
		return byteStats[i].count > byteStats[j].count
	})
	for i, stat := range byteStats {
		if i >= 8 {
			break
		}
		analysis.HotBytes = append(analysis.HotBytes, model.DoctorCheck{
			Name:    fmt.Sprintf("byte-%02d", stat.index),
			Status:  fmt.Sprintf("%d", stat.count),
			Details: "unique values",
		})
	}

	type kv struct {
		k string
		v int
	}
	var ops []kv
	for k, v := range counts {
		ops = append(ops, kv{k, v})
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].v > ops[j].v })
	for i, item := range ops {
		if i >= 8 {
			break
		}
		analysis.TopOperations = append(analysis.TopOperations, model.DoctorCheck{Name: item.k, Status: fmt.Sprintf("%d", item.v)})
	}
	return analysis, nil
}

func BuildTXIDAnalysisReport(filePath string, operationFilter string, includeBitMap bool) (model.DoctorReport, error) {
	analysis, err := AnalyzeTXIDTrace(filePath, operationFilter)
	if err != nil {
		return model.DoctorReport{}, err
	}
	title := "TXID Analysis"
	if analysis.OperationFilter != "" {
		title += " [" + analysis.OperationFilter + "]"
	}
	checks := []model.DoctorCheck{
		{Name: "samples", Status: fmt.Sprintf("%d", analysis.Samples)},
		{Name: "operations", Status: fmt.Sprintf("%d", analysis.Operations)},
		{Name: "encoded-length", Status: fmt.Sprintf("%d-%d", analysis.MinEncodedLength, analysis.MaxEncodedLength)},
		{Name: "decoded-length", Status: fmt.Sprintf("%d-%d", analysis.MinDecodedLength, analysis.MaxDecodedLength)},
		{Name: "avg-decoded-bytes", Status: fmt.Sprintf("%.1f", analysis.AverageDecodedLen)},
		{Name: "unique-salts", Status: fmt.Sprintf("%d", analysis.UniqueSalts), Details: "distinct salt patterns detected"},
		{Name: "same-salt-ops", Status: fmt.Sprintf("%d", analysis.SameSaltOps), Details: "operations sharing same salt"},
		{Name: "avg-bit-delta", Status: fmt.Sprintf("%.1f", analysis.AverageBitDelta), Details: "same-operation consecutive samples"},
		{Name: "same-second-pairs", Status: fmt.Sprintf("%d", analysis.SameSecondPairs)},
		{Name: "same-second-bit-delta", Status: fmt.Sprintf("%.1f", analysis.SameSecondAvgDelta), Details: "same-operation within same second"},
		{Name: "same-ms-pairs", Status: fmt.Sprintf("%d", analysis.SameMSPairs)},
		{Name: "same-ms-bit-delta", Status: fmt.Sprintf("%.1f", analysis.SameMSAvgDelta), Details: "same-operation within same millisecond"},
		{Name: "ct0-hash-variants", Status: fmt.Sprintf("%d", analysis.CT0Variants)},
		{Name: "auth-hash-variants", Status: fmt.Sprintf("%d", analysis.AuthVariants)},
		{Name: "stable-bits", Status: fmt.Sprintf("%d", analysis.StableBits)},
		{Name: "biased-bits", Status: fmt.Sprintf("%d", analysis.BiasedBits), Details: "P(1) <= 0.1 or >= 0.9"},
		{Name: "stable-prefix-bytes", Status: fmt.Sprintf("%d", analysis.StablePrefixBytes)},
		{Name: "stable-byte-count", Status: fmt.Sprintf("%d", analysis.StableByteCount)},
	}
	if includeBitMap {
		for i, prob := range analysis.BitProbabilities {
			checks = append(checks, model.DoctorCheck{
				Name:    fmt.Sprintf("bit-%03d", i),
				Status:  fmt.Sprintf("%.3f", prob),
				Details: "P(1)",
			})
		}
	}
	checks = append(checks, analysis.FirstBytes...)
	checks = append(checks, analysis.HotBytes...)
	checks = append(checks, analysis.TopOperations...)
	checks = append(checks, analysis.SaltPatterns...)
	return model.DoctorReport{Name: title, Checks: checks, Transport: filePath, Auth: time.Now().UTC().Format(time.RFC3339)}, nil
}

func padBase64(value string) string {
	if rem := len(value) % 4; rem != 0 {
		value += strings.Repeat("=", 4-rem)
	}
	return value
}

func bitHammingDistance(a []byte, b []byte) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	distance := 0
	for i := 0; i < limit; i++ {
		distance += bitsInByte(a[i] ^ b[i])
	}
	if len(a) > limit {
		for _, v := range a[limit:] {
			distance += bitsInByte(v)
		}
	}
	if len(b) > limit {
		for _, v := range b[limit:] {
			distance += bitsInByte(v)
		}
	}
	return distance
}

func bitsInByte(v byte) int {
	count := 0
	for v != 0 {
		count += int(v & 1)
		v >>= 1
	}
	return count
}

func ExtractSalts(filePath string) ([]model.SaltSample, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var samples []model.SaltSample
	seen := map[string]bool{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec txIDTraceRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}

		salt := rec.TxIDSalt
		if salt == "" && rec.TxID != "" {
			if decoded, err := base64.StdEncoding.DecodeString(padBase64(rec.TxID)); err == nil && len(decoded) >= 70 {
				salt = fmt.Sprintf("%x", decoded[41:70])
			}
		}
		if salt == "" {
			continue
		}

		key := rec.Operation + "|" + salt
		if seen[key] {
			continue
		}
		seen[key] = true

		samples = append(samples, model.SaltSample{
			Operation:   rec.Operation,
			TimestampMS: rec.TimestampMS,
			Salt:        salt,
			FullTxID:    rec.TxID,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(samples, func(i, j int) bool {
		if samples[i].Operation != samples[j].Operation {
			return samples[i].Operation < samples[j].Operation
		}
		return samples[i].TimestampMS < samples[j].TimestampMS
	})

	return samples, nil
}

func CompareSalts(filePath string) ([]model.SaltComparison, error) {
	samples, err := ExtractSalts(filePath)
	if err != nil {
		return nil, err
	}

	var comparisons []model.SaltComparison
	for i := 0; i < len(samples)-1; i++ {
		for j := i + 1; j < len(samples); j++ {
			a, b := samples[i], samples[j]
			gap := (b.TimestampMS - a.TimestampMS) / 1000

			comparisons = append(comparisons, model.SaltComparison{
				SampleA:    a,
				SampleB:    b,
				SaltMatch:  a.Salt == b.Salt,
				TimeGapSec: gap,
				SameOp:     a.Operation == b.Operation,
			})
		}
	}

	return comparisons, nil
}

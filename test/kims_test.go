package test

import (
	"fmt"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"testing"
	"time"

	"github.com/bnb-chain/zkbnb-setup/phase1"
)

func TestSetup_kims(t *testing.T) {

	var power byte = 11 //随着power增加约束数量增加，contribute时间增加
	// 9-120ms 11-430ms in macIntel

	time1 := time.Now()
	// Initialize to Phase 1
	if err := phase1.Initialize(power, "0.ph1"); err != nil {
		t.Error(err)
	}
	fmt.Printf("Initialize time: %v\n", time.Now().Sub(time1))

	for i := 0; i < 10; i++ {
		fmt.Println("————————————————————————————")
		from := fmt.Sprintf("%d.ph1", i)
		to := fmt.Sprintf("%d.ph1", i+1)

		var tau, alpha, beta, one fr.Element
		tau.SetRandom()
		alpha.SetRandom()
		beta.SetRandom()
		one.SetOne()
		fmt.Println("tau:", tau.String())
		fmt.Println("alpha:", alpha.String())
		fmt.Println("beta:", beta.String())
		//fmt.Println("one:", one.String())

		start := time.Now()
		contrihash, err := phase1.ContributeServer(from, to, tau, alpha, beta, one)
		if err != nil {
			t.Error(err)
			return // 出现错误时退出
		}
		// 计算并打印时间
		duration := time.Since(start)
		fmt.Printf("Contributed from %s to %s in %v\n", from, to, duration)
		fmt.Println("Contribution Hash := ", contrihash)
	}

	// Verify Phase 1 contributions
	if err := phase1.Verify("10.ph1", ""); err != nil {
		t.Error(err)
	}
	//
	//// Phase 2 initialization
	//if err := phase2.Initialize("4.ph1", "circuit.r1cs", "0.ph2"); err != nil {
	//	t.Error(err)
	//}
	//
	//// Contribute to Phase 2
	//if err := phase2.Contribute("0.ph2", "1.ph2"); err != nil {
	//	t.Error(err)
	//}
	//
	//if err := phase2.Contribute("1.ph2", "2.ph2"); err != nil {
	//	t.Error(err)
	//}
	//
	//if err := phase2.Contribute("2.ph2", "3.ph2"); err != nil {
	//	t.Error(err)
	//}
	//
	//// Verify Phase 2 contributions
	//if err := phase2.Verify("3.ph2", "0.ph2"); err != nil {
	//	t.Error(err)
	//}
	//
	//if err := keys.ExtractKeys("3.ph2"); err != nil {
	//	t.Error(err)
	//}
}

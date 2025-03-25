package kims_test

import (
	"fmt"
	"github.com/bnb-chain/zkbnb-setup/phase1"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/rs/zerolog/log"
	"testing"
	"time"
)

func Test_main(t *testing.T) {
	test_()
	//test_10_origin()
}

func test_() {

	for i := 0; i < 10; i++ {
		fmt.Println("————————————————————————————")
		from := fmt.Sprintf("%d.ph1", i)
		to := fmt.Sprintf("%d.ph1", i+1)

		var tau, alpha, beta, one fr.Element
		tau.SetString("5349450036908408402062633344441795636900283135072291564543572822568951905530")
		alpha.SetString("10309324653435217550478051930870500767202400222442521321085187176798090049296")
		beta.SetString("3876027126774557149913561148679727244308876819583237066015246325773858351512")
		one.SetOne()
		fmt.Println("tau:", tau.String())
		fmt.Println("alpha:", alpha.String())
		fmt.Println("beta:", beta.String())
		//fmt.Println("one:", one.String())

		start := time.Now()
		//contrihash, err := phase1.ContributeAccelerate(from, to, tau, alpha, beta, one)
		contrihash, err := phase1.ContributeServer(from, to, tau, alpha, beta, one)
		if err != nil {
			log.Error().Msgf("Contribute error: %s", err.Error())
			return // 出现错误时退出
		}
		// 计算并打印时间
		duration := time.Since(start)
		fmt.Printf("Contributed from %s to %s in %v\n", from, to, duration)
		fmt.Println("Contribution Hash := ", contrihash)

		// Verify Phase 1 contributions
		if err := phase1.Verify_(fmt.Sprintf("%d.ph1", i+1), ""); err != nil {
			log.Error().Msgf("Verification error: %s", err.Error())
		}
	}
}

func test_10_origin() {

	for i := 0; i < 10; i++ {
		fmt.Println("————————————————————————————")
		from := fmt.Sprintf("%d.ph1", i)
		to := fmt.Sprintf("%d.ph1", i+1)

		var tau, alpha, beta, one fr.Element
		tau.SetString("5349450036908408402062633344441795636900283135072291564543572822568951905530")
		alpha.SetString("10309324653435217550478051930870500767202400222442521321085187176798090049296")
		beta.SetString("3876027126774557149913561148679727244308876819583237066015246325773858351512")
		one.SetOne()
		fmt.Println("tau:", tau.String())
		fmt.Println("alpha:", alpha.String())
		fmt.Println("beta:", beta.String())

		start := time.Now()
		contrihash, err := phase1.ContributeServerAll(from, to, tau, alpha, beta, one)
		if err != nil {
			log.Error().Msgf("ContributeServerAll: %s", err.Error())
			return // 出现错误时退出
		}
		// 计算并打印时间
		duration := time.Since(start)
		fmt.Printf("Contributed from %s to %s in %v\n", from, to, duration)
		fmt.Println("Contribution Hash := ", contrihash)
	}

	// Verify Phase 1 contributions
	if err := phase1.Verify("10.ph1", ""); err != nil {
		log.Error().Msgf("Verification error: %s", err.Error())
	}
}

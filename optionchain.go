package trade

import (
	"fmt"
	"time"
)

type OptionChains map[time.Time]*OptionChain

type OptionChain struct {
	Expiry  time.Time
	Strikes map[float64]*Strike
}

type Strike struct {
	Price float64
	Put   *Option
	Call  *Option
}

func (engine *Engine) GetOptionChains(spot Instrument, unknown chan interface{}) (OptionChains, error) {
	req := spot.ContractDataReq()
	req.SecurityType = "OPT"
	engine.In <- req

	// temporary option chains
	chains := make(OptionChains)

	var v interface{}

done:

	// message loop
	for {
		select {
		case <-time.After(30 * time.Second):
			return nil, timeout()
		case v = <-engine.Out:
		case err := <-engine.Error:
			return nil, err
		}

		switch v.(type) {
		case *ContractDataEnd:
			break done
		case *ContractData:
			v := v.(*ContractData)
			expiry, err := time.Parse("20060102", v.Expiry)
			if err != nil {
				return nil, err
			}
			if chain, ok := chains[expiry]; ok {
				chain.update(v)
			} else {
				chain := &OptionChain{
					Expiry:  expiry,
					Strikes: make(map[float64]*Strike),
				}
				chain.update(v)
				chains[expiry] = chain
			}
		default:
			unknown <- v
		}
	}

	return chains, nil
}

func (strike *Strike) update(v *ContractData) {
	option := &Option{
		ContractId: v.ContractId,
	}

	if v.Right == "C" {
		option.Type = CALL
		strike.Call = option
	} else {
		option.Type = PUT
		strike.Put = option
	}
}

func (chain *OptionChain) update(v *ContractData) {
	if strike, ok := chain.Strikes[v.Strike]; ok {
		// strike exists
		strike.update(v)
	} else {
		// no strike exists
		strike := &Strike{
			Price: v.Strike,
		}
		chain.Strikes[v.Strike] = strike
		strike.update(v)
	}
}

func (strike *Strike) String() string {
	toString := func(v *Option, label string) string {
		if v == nil {
			return ""
		}

		return fmt.Sprintf("%s %d", label, v.ContractId)
	}

	options := toString(strike.Call, "CALL") + " " + toString(strike.Put, "PUT")

	return fmt.Sprintf("%.5g %s", strike.Price, options)
}

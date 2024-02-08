package network

import (
	"btc-indexer/pkg/logger"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
)

var log = logger.NewDefaultLogger()

func LookUpPeers(DNSSeeds []chaincfg.DNSSeed, defaultPort uint16, peerIpChan chan *wire.NetAddressV2) {

	wg := new(sync.WaitGroup)
	for _, dnsseed := range DNSSeeds {
		go func(host string) {
			defer wg.Done()
			seedpeers, err := net.LookupIP(host)
			if err != nil {
				log.Warn(err.Error())
			}

			log.Info(fmt.Sprintf("Found %d Peers From %s", len(seedpeers), host))

			for _, seedpeer := range seedpeers {
				peerIpChan <- wire.NetAddressV2FromBytes(
					time.Now().Add(-1*time.Second*time.Duration(24*60*60*3)), // marking address vs seen 3 days before,
					0, seedpeer, defaultPort,
				)
			}
		}(dnsseed.Host)
		wg.Add(1)
	}
	wg.Wait()
	close(peerIpChan)
}

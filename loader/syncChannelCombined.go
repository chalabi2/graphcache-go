package loader

import (
	"encoding/json"
	"log"
)

type combinedEntry struct {
	Block struct {
		Time   int    `json:"timestamp"`
		Number int    `json:"number"`
		Hash   string `json:"hash"`
	} `json:"block"`
}

type CombinedData struct {
	Meta  combinedEntry   `json:"_meta"`
	Swaps json.RawMessage `json:"swaps"`
	Aggs  json.RawMessage `json:"aggEvents"`
	Liqs  json.RawMessage `json:"liquidityChanges"`
	Kos   json.RawMessage `json:"knockoutCrosses"`
	Fees  json.RawMessage `json:"feeChanges"`
	Bals  json.RawMessage `json:"userBalances"`
}

type combinedWrapper struct {
	Data CombinedData `json:"data"`
}

type CombinedStartBlocks struct {
	Swap int
	Agg  int
	Bal  int
	Fee  int
	Ko   int
	Liq  int
}

func CombinedQuery(cfg SyncChannelConfig, minBlocks CombinedStartBlocks, maxBlock int) (metaBlock int, combinedData *CombinedData, err error) {
	cfg.Query = "./artifacts/graphQueries/combined.query"
	combinedQuery := readQueryPath(cfg.Query)

	resp, err := queryFromSubgraphCombined(cfg.Chain, combinedQuery, true, minBlocks, maxBlock)
	if err != nil {
		return 0, nil, err
	}

	result, err := parseCombinedResp(resp)
	if err != nil {
		return 0, nil, err
	}

	if result.Meta.Block.Number == 0 {
		log.Println("Warning subgraph latest block number is 0. Retrying ", cfg.Network)
		return CombinedQuery(cfg, minBlocks, maxBlock)
	} else {
		return result.Meta.Block.Number, result, nil
	}
}

func parseCombinedResp(body []byte) (*CombinedData, error) {
	var parsed combinedWrapper

	err := json.Unmarshal(body, &parsed)
	if err != nil {
		return nil, err
	}
	return &parsed.Data, nil
}

func (s *SyncChannel[R, S]) IngestEntries(data []byte, queryStartBlock, queryEndBlock int) (lastObs int, hasMore bool, err error) {
	nIngested := 0
	lastObs = queryStartBlock

	entries, err := s.tbl.ParseSubGraphRespUnwrapped(data)
	if err != nil {
		log.Println("Warning subgraph data decode error: " + err.Error())
		return 0, true, err
	}

	if len(entries) == 0 {
		log.Printf("Warning subgraph data for %s returned no entries while the last seen row was expected", s.config.Query)
		return 0, true, nil
	}

	for _, entry := range entries {
		row := s.tbl.ConvertSubGraphRow(entry, string(s.config.Network))
		isFreshPoint, eventBlock := s.ingestEntry(row)

		if isFreshPoint {
			nIngested += 1
		}
		if eventBlock > lastObs {
			lastObs = eventBlock
		}
	}

	if nIngested > 0 {
		log.Printf("Loaded %d rows (total %d) from subgraph from query %s on block=%d-%d",
			nIngested, s.RowsIngested, s.config.Query, queryStartBlock, queryEndBlock)
	}
	hasMore = nIngested > 0 || len(entries) == 1000
	// log.Println("Ingested", nIngested, "rows, len", len(entries), "hasMore", hasMore, s.config.Query)
	return lastObs, hasMore, nil
}

# Phase 12.0 - Edge Discovery Matrix

## Summary
- final_label: `robust_edges_found`
- measurement cadence: `15m fixed grid over existing 1m candle/features rows`
- coverage achieved: `192/192`
- ak_trader_touched: `false`
- promotion_allowed: `false`
- raw_required: `false`
- raw_event_detail_retained: `false`
- bucket_count: `2772`
- rejected bucket count: `2388`
- data-insufficient bucket count: `300`
- weak bucket count: `71`
- robust bucket count: `13`
- strongest bucket: `RealizedVol60/realized_vol_60_mid/trend_down/long/240m` exp5=`16.994585` PF=`1.174282` label=`ROBUST_EDGE_CANDIDATE`
- worth turning into a strategy: `true`
- recommended next phase: `Phase 12.1 - Convert Top Robust Edge Bucket Into Candidate Family`

## Feature Inventory
- Return15: available=`true`, source=`features.Row.Return15`, buckets=`return15_down, return15_flat, return15_up`
- TrendSlope20: available=`true`, source=`features.Row.TrendSlope20`, buckets=`slope_down, slope_flat, slope_up`
- CloseRelativeEMA20: available=`true`, source=`features.Row.Close and EMA20`, buckets=`below_ema20, near_ema20, above_ema20`
- BBWidthPctRank60: available=`true`, source=`features.Row.BBWidthPctRank60`, buckets=`bb_width_low, bb_width_mid, bb_width_high`
- VolumeRatio20: available=`true`, source=`features.Row.VolumeRatio20`, buckets=`volume_ratio_low, volume_ratio_normal, volume_ratio_high`
- TakerBuyRatio: available=`true`, source=`features.Row.TakerBuyRatio`, buckets=`taker_sell_heavy, taker_balanced, taker_buy_heavy`
- CompositeRegime: available=`true`, source=`derived from existing features`, buckets=`compressed_down, compressed_up, trend_down, trend_up, range`
- BTCContext60: available=`true`, source=`features.Row.BTCReturn60`, buckets=`btc_down, btc_flat, btc_up`
- ETHContext60: available=`true`, source=`features.Row.ETHReturn60`, buckets=`eth_down, eth_flat, eth_up`
- SessionUTC: available=`true`, source=`derived from event_time_ms`, buckets=`asia, europe, us, late_us`
- ATRPct14: available=`true`, source=`features.Row.ATRPct14`, buckets=`atr_pct_14_low, atr_pct_14_mid, atr_pct_14_high`
- RealizedVol60: available=`true`, source=`features.Row.RealizedVol60`, buckets=`realized_vol_60_low, realized_vol_60_mid, realized_vol_60_high`

## Top 20 By Expectancy After 5 bps
| Feature | Bucket | Regime | Side | Horizon | Label | Samples | Clusters | Exp5 | Exp7.5 | Exp10 | PF | Win% | Worst Month | Worst Symbol | Top Symbol% | Top Month% |
|---|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| RealizedVol60 | realized_vol_60_high | trend_up | long | 240m | DATA_INSUFFICIENT | 90 | 59 | 246.187300 | 243.687300 | 241.187300 | 3.435526 | 68.8889 | -386.485674 | -72.728668 | 36.1135 | 55.3724 |
| RealizedVol60 | realized_vol_60_high | trend_down | long | 240m | DATA_INSUFFICIENT | 258 | 154 | 245.775697 | 243.275697 | 240.775697 | 3.500747 | 74.4186 | -146.298433 | 116.653382 | 20.7549 | 28.3847 |
| RealizedVol60 | realized_vol_60_unknown | range | long | 240m | DATA_INSUFFICIENT | 7 | 7 | 219.986245 | 217.486245 | 214.986245 | 999.000000 | 100.0000 | 219.986245 | 64.605568 | 34.4985 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_down | short | 120m | DATA_INSUFFICIENT | 8 | 8 | 207.779075 | 205.279075 | 202.779075 | 999.000000 | 100.0000 | 207.779075 | 102.203375 | 20.9950 | 100.0000 |
| RealizedVol60 | realized_vol_60_unknown | trend_up | long | 120m | DATA_INSUFFICIENT | 1 | 1 | 207.733039 | 205.233039 | 202.733039 | 999.000000 | 100.0000 | 207.733039 | 207.733039 | 100.0000 | 100.0000 |
| RealizedVol60 | realized_vol_60_high | trend_down | long | 120m | DATA_INSUFFICIENT | 258 | 154 | 205.262584 | 202.762584 | 200.262584 | 4.132423 | 68.6047 | -63.577017 | 126.494189 | 20.4669 | 27.1482 |
| ATRPct14 | atr_pct_14_unknown | range | long | 240m | DATA_INSUFFICIENT | 17 | 7 | 197.790311 | 195.290311 | 192.790311 | 999.000000 | 100.0000 | 197.790311 | 53.971384 | 29.8351 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_down | short | 240m | DATA_INSUFFICIENT | 8 | 8 | 192.203070 | 189.703070 | 187.203070 | 999.000000 | 100.0000 | 192.203070 | 36.152263 | 24.0352 | 100.0000 |
| RealizedVol60 | realized_vol_60_high | trend_up | long | 120m | DATA_INSUFFICIENT | 90 | 59 | 187.580963 | 185.080963 | 182.580963 | 2.644576 | 60.0000 | -312.484568 | -35.327978 | 38.0334 | 52.9256 |
| RealizedVol60 | realized_vol_60_high | trend_up | long | 60m | DATA_INSUFFICIENT | 90 | 59 | 183.775002 | 181.275002 | 178.775002 | 2.956625 | 65.5556 | -299.735290 | -15.792708 | 40.5182 | 55.2306 |
| ATRPct14 | atr_pct_14_high | compressed_up | long | 240m | DATA_INSUFFICIENT | 199 | 135 | 182.588154 | 180.088154 | 177.588154 | 2.509258 | 66.8342 | -145.127284 | -28.458872 | 29.5207 | 34.8449 |
| RealizedVol60 | realized_vol_60_unknown | trend_up | long | 240m | DATA_INSUFFICIENT | 1 | 1 | 179.497563 | 176.997563 | 174.497563 | 999.000000 | 100.0000 | 179.497563 | 179.497563 | 100.0000 | 100.0000 |
| CloseRelativeEMA20 | above_ema20_strong | trend_down | long | 240m | DATA_INSUFFICIENT | 107 | 104 | 179.382642 | 176.882642 | 174.382642 | 3.237623 | 67.2897 | -196.123551 | -84.694354 | 36.6867 | 29.4174 |
| RealizedVol60 | realized_vol_60_high | compressed_up | long | 240m | DATA_INSUFFICIENT | 136 | 90 | 178.477442 | 175.977442 | 173.477442 | 2.773586 | 65.4412 | -322.463904 | 16.108036 | 38.4782 | 48.8994 |
| VolumeRatio20 | volume_ratio_unknown | range | long | 240m | DATA_INSUFFICIENT | 22 | 12 | 170.985633 | 168.485633 | 165.985633 | 999.000000 | 100.0000 | 170.985633 | 45.915184 | 30.8961 | 100.0000 |
| RealizedVol60 | realized_vol_60_unknown | trend_up | long | 60m | DATA_INSUFFICIENT | 1 | 1 | 169.827880 | 167.327880 | 164.827880 | 999.000000 | 100.0000 | 169.827880 | 169.827880 | 100.0000 | 100.0000 |
| ATRPct14 | atr_pct_14_high | compressed_down | long | 240m | DATA_INSUFFICIENT | 225 | 187 | 161.203899 | 158.703899 | 156.203899 | 2.128742 | 59.5556 | -250.622282 | 10.678161 | 38.0111 | 28.1978 |
| VolumeRatio20 | volume_ratio_unknown | compressed_down | long | 240m | DATA_INSUFFICIENT | 8 | 8 | 153.920364 | 151.420364 | 148.920364 | 999.000000 | 100.0000 | 153.920364 | 29.802784 | 30.4880 | 100.0000 |
| ATRPct14 | atr_pct_14_unknown | compressed_down | long | 240m | DATA_INSUFFICIENT | 8 | 8 | 153.920364 | 151.420364 | 148.920364 | 999.000000 | 100.0000 | 153.920364 | 29.802784 | 30.4880 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_up | long | 120m | DATA_INSUFFICIENT | 7 | 5 | 151.865827 | 149.365827 | 146.865827 | 999.000000 | 100.0000 | 151.865827 | 60.738592 | 56.5856 | 100.0000 |

## Top 20 By Expectancy After 7.5 bps
| Feature | Bucket | Regime | Side | Horizon | Label | Samples | Clusters | Exp5 | Exp7.5 | Exp10 | PF | Win% | Worst Month | Worst Symbol | Top Symbol% | Top Month% |
|---|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| RealizedVol60 | realized_vol_60_high | trend_up | long | 240m | DATA_INSUFFICIENT | 90 | 59 | 246.187300 | 243.687300 | 241.187300 | 3.435526 | 68.8889 | -386.485674 | -72.728668 | 36.1135 | 55.3724 |
| RealizedVol60 | realized_vol_60_high | trend_down | long | 240m | DATA_INSUFFICIENT | 258 | 154 | 245.775697 | 243.275697 | 240.775697 | 3.500747 | 74.4186 | -146.298433 | 116.653382 | 20.7549 | 28.3847 |
| RealizedVol60 | realized_vol_60_unknown | range | long | 240m | DATA_INSUFFICIENT | 7 | 7 | 219.986245 | 217.486245 | 214.986245 | 999.000000 | 100.0000 | 219.986245 | 64.605568 | 34.4985 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_down | short | 120m | DATA_INSUFFICIENT | 8 | 8 | 207.779075 | 205.279075 | 202.779075 | 999.000000 | 100.0000 | 207.779075 | 102.203375 | 20.9950 | 100.0000 |
| RealizedVol60 | realized_vol_60_unknown | trend_up | long | 120m | DATA_INSUFFICIENT | 1 | 1 | 207.733039 | 205.233039 | 202.733039 | 999.000000 | 100.0000 | 207.733039 | 207.733039 | 100.0000 | 100.0000 |
| RealizedVol60 | realized_vol_60_high | trend_down | long | 120m | DATA_INSUFFICIENT | 258 | 154 | 205.262584 | 202.762584 | 200.262584 | 4.132423 | 68.6047 | -63.577017 | 126.494189 | 20.4669 | 27.1482 |
| ATRPct14 | atr_pct_14_unknown | range | long | 240m | DATA_INSUFFICIENT | 17 | 7 | 197.790311 | 195.290311 | 192.790311 | 999.000000 | 100.0000 | 197.790311 | 53.971384 | 29.8351 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_down | short | 240m | DATA_INSUFFICIENT | 8 | 8 | 192.203070 | 189.703070 | 187.203070 | 999.000000 | 100.0000 | 192.203070 | 36.152263 | 24.0352 | 100.0000 |
| RealizedVol60 | realized_vol_60_high | trend_up | long | 120m | DATA_INSUFFICIENT | 90 | 59 | 187.580963 | 185.080963 | 182.580963 | 2.644576 | 60.0000 | -312.484568 | -35.327978 | 38.0334 | 52.9256 |
| RealizedVol60 | realized_vol_60_high | trend_up | long | 60m | DATA_INSUFFICIENT | 90 | 59 | 183.775002 | 181.275002 | 178.775002 | 2.956625 | 65.5556 | -299.735290 | -15.792708 | 40.5182 | 55.2306 |
| ATRPct14 | atr_pct_14_high | compressed_up | long | 240m | DATA_INSUFFICIENT | 199 | 135 | 182.588154 | 180.088154 | 177.588154 | 2.509258 | 66.8342 | -145.127284 | -28.458872 | 29.5207 | 34.8449 |
| RealizedVol60 | realized_vol_60_unknown | trend_up | long | 240m | DATA_INSUFFICIENT | 1 | 1 | 179.497563 | 176.997563 | 174.497563 | 999.000000 | 100.0000 | 179.497563 | 179.497563 | 100.0000 | 100.0000 |
| CloseRelativeEMA20 | above_ema20_strong | trend_down | long | 240m | DATA_INSUFFICIENT | 107 | 104 | 179.382642 | 176.882642 | 174.382642 | 3.237623 | 67.2897 | -196.123551 | -84.694354 | 36.6867 | 29.4174 |
| RealizedVol60 | realized_vol_60_high | compressed_up | long | 240m | DATA_INSUFFICIENT | 136 | 90 | 178.477442 | 175.977442 | 173.477442 | 2.773586 | 65.4412 | -322.463904 | 16.108036 | 38.4782 | 48.8994 |
| VolumeRatio20 | volume_ratio_unknown | range | long | 240m | DATA_INSUFFICIENT | 22 | 12 | 170.985633 | 168.485633 | 165.985633 | 999.000000 | 100.0000 | 170.985633 | 45.915184 | 30.8961 | 100.0000 |
| RealizedVol60 | realized_vol_60_unknown | trend_up | long | 60m | DATA_INSUFFICIENT | 1 | 1 | 169.827880 | 167.327880 | 164.827880 | 999.000000 | 100.0000 | 169.827880 | 169.827880 | 100.0000 | 100.0000 |
| ATRPct14 | atr_pct_14_high | compressed_down | long | 240m | DATA_INSUFFICIENT | 225 | 187 | 161.203899 | 158.703899 | 156.203899 | 2.128742 | 59.5556 | -250.622282 | 10.678161 | 38.0111 | 28.1978 |
| VolumeRatio20 | volume_ratio_unknown | compressed_down | long | 240m | DATA_INSUFFICIENT | 8 | 8 | 153.920364 | 151.420364 | 148.920364 | 999.000000 | 100.0000 | 153.920364 | 29.802784 | 30.4880 | 100.0000 |
| ATRPct14 | atr_pct_14_unknown | compressed_down | long | 240m | DATA_INSUFFICIENT | 8 | 8 | 153.920364 | 151.420364 | 148.920364 | 999.000000 | 100.0000 | 153.920364 | 29.802784 | 30.4880 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_up | long | 120m | DATA_INSUFFICIENT | 7 | 5 | 151.865827 | 149.365827 | 146.865827 | 999.000000 | 100.0000 | 151.865827 | 60.738592 | 56.5856 | 100.0000 |

## Top 20 By Profit Factor
| Feature | Bucket | Regime | Side | Horizon | Label | Samples | Clusters | Exp5 | Exp7.5 | Exp10 | PF | Win% | Worst Month | Worst Symbol | Top Symbol% | Top Month% |
|---|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| VolumeRatio20 | volume_ratio_unknown | range | long | 240m | DATA_INSUFFICIENT | 22 | 12 | 170.985633 | 168.485633 | 165.985633 | 999.000000 | 100.0000 | 170.985633 | 45.915184 | 30.8961 | 100.0000 |
| ATRPct14 | atr_pct_14_unknown | range | long | 240m | DATA_INSUFFICIENT | 17 | 7 | 197.790311 | 195.290311 | 192.790311 | 999.000000 | 100.0000 | 197.790311 | 53.971384 | 29.8351 | 100.0000 |
| ATRPct14 | atr_pct_14_unknown | range | long | 60m | DATA_INSUFFICIENT | 17 | 7 | 89.444803 | 86.944803 | 84.444803 | 999.000000 | 100.0000 | 89.444803 | 36.569992 | 24.6365 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_down | short | 120m | DATA_INSUFFICIENT | 8 | 8 | 207.779075 | 205.279075 | 202.779075 | 999.000000 | 100.0000 | 207.779075 | 102.203375 | 20.9950 | 100.0000 |
| ATRPct14 | atr_pct_14_unknown | compressed_down | long | 240m | DATA_INSUFFICIENT | 8 | 8 | 153.920364 | 151.420364 | 148.920364 | 999.000000 | 100.0000 | 153.920364 | 29.802784 | 30.4880 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_down | short | 240m | DATA_INSUFFICIENT | 8 | 8 | 192.203070 | 189.703070 | 187.203070 | 999.000000 | 100.0000 | 192.203070 | 36.152263 | 24.0352 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | compressed_down | long | 240m | DATA_INSUFFICIENT | 8 | 8 | 153.920364 | 151.420364 | 148.920364 | 999.000000 | 100.0000 | 153.920364 | 29.802784 | 30.4880 | 100.0000 |
| ATRPct14 | atr_pct_14_unknown | compressed_down | long | 120m | DATA_INSUFFICIENT | 8 | 8 | 142.722517 | 140.222517 | 137.722517 | 999.000000 | 100.0000 | 142.722517 | 47.204176 | 22.1034 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | compressed_down | long | 120m | DATA_INSUFFICIENT | 8 | 8 | 142.722517 | 140.222517 | 137.722517 | 999.000000 | 100.0000 | 142.722517 | 47.204176 | 22.1034 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_down | short | 60m | DATA_INSUFFICIENT | 8 | 8 | 148.040785 | 145.540785 | 143.040785 | 999.000000 | 100.0000 | 148.040785 | 79.379431 | 22.3781 | 100.0000 |
| RealizedVol60 | realized_vol_60_unknown | range | long | 15m | DATA_INSUFFICIENT | 7 | 7 | 41.642159 | 39.142159 | 36.642159 | 999.000000 | 100.0000 | 41.642159 | 0.800464 | 31.3981 | 100.0000 |
| RealizedVol60 | realized_vol_60_unknown | range | long | 60m | DATA_INSUFFICIENT | 7 | 7 | 120.549083 | 118.049083 | 115.549083 | 999.000000 | 100.0000 | 120.549083 | 41.403712 | 21.3784 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_up | long | 60m | DATA_INSUFFICIENT | 7 | 5 | 69.749390 | 67.249390 | 64.749390 | 999.000000 | 100.0000 | 69.749390 | 0.800464 | 63.5516 | 100.0000 |
| ATRPct14 | atr_pct_14_unknown | trend_up | long | 60m | DATA_INSUFFICIENT | 7 | 5 | 69.749390 | 67.249390 | 64.749390 | 999.000000 | 100.0000 | 69.749390 | 0.800464 | 63.5516 | 100.0000 |
| RealizedVol60 | realized_vol_60_unknown | range | long | 120m | DATA_INSUFFICIENT | 7 | 7 | 121.068439 | 118.568439 | 116.068439 | 999.000000 | 100.0000 | 121.068439 | 13.295376 | 30.9997 | 100.0000 |
| ATRPct14 | atr_pct_14_unknown | trend_up | long | 240m | DATA_INSUFFICIENT | 7 | 5 | 148.246491 | 145.746491 | 143.246491 | 999.000000 | 100.0000 | 148.246491 | 31.736272 | 49.4316 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_up | long | 120m | DATA_INSUFFICIENT | 7 | 5 | 151.865827 | 149.365827 | 146.865827 | 999.000000 | 100.0000 | 151.865827 | 60.738592 | 56.5856 | 100.0000 |
| VolumeRatio20 | volume_ratio_unknown | trend_up | long | 240m | DATA_INSUFFICIENT | 7 | 5 | 148.246491 | 145.746491 | 143.246491 | 999.000000 | 100.0000 | 148.246491 | 31.736272 | 49.4316 | 100.0000 |
| ATRPct14 | atr_pct_14_unknown | trend_up | long | 120m | DATA_INSUFFICIENT | 7 | 5 | 151.865827 | 149.365827 | 146.865827 | 999.000000 | 100.0000 | 151.865827 | 60.738592 | 56.5856 | 100.0000 |
| RealizedVol60 | realized_vol_60_unknown | range | long | 240m | DATA_INSUFFICIENT | 7 | 7 | 219.986245 | 217.486245 | 214.986245 | 999.000000 | 100.0000 | 219.986245 | 64.605568 | 34.4985 | 100.0000 |

## Robust Edge Candidate Buckets
| Feature | Bucket | Regime | Side | Horizon | Label | Samples | Clusters | Exp5 | Exp7.5 | Exp10 | PF | Win% | Worst Month | Worst Symbol | Top Symbol% | Top Month% |
|---|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| RealizedVol60 | realized_vol_60_mid | trend_down | long | 240m | ROBUST_EDGE_CANDIDATE | 18013 | 9618 | 16.994585 | 14.494585 | 11.994585 | 1.174282 | 54.9936 | -42.937228 | -7.524183 | 23.8566 | 23.5199 |
| ETHContext60 | eth_down | compressed_up | short | 240m | ROBUST_EDGE_CANDIDATE | 1354 | 1292 | 15.773165 | 13.273165 | 10.773165 | 1.220991 | 53.6189 | -94.943953 | -4.227680 | 29.1235 | 20.7340 |
| CloseRelativeEMA20 | below_ema20_strong | range | long | 240m | ROBUST_EDGE_CANDIDATE | 6205 | 5393 | 14.331404 | 11.831404 | 9.331404 | 1.159586 | 51.5230 | -52.446935 | 3.400412 | 22.2300 | 32.8838 |
| CloseRelativeEMA20 | above_ema20_strong | range | long | 240m | ROBUST_EDGE_CANDIDATE | 7404 | 6323 | 13.711305 | 11.211305 | 8.711305 | 1.142121 | 52.4041 | -98.136110 | 6.031416 | 20.7756 | 25.3884 |
| RealizedVol60 | realized_vol_60_mid | compressed_up | long | 240m | ROBUST_EDGE_CANDIDATE | 9772 | 6723 | 12.784558 | 10.284558 | 7.784558 | 1.138915 | 51.4429 | -66.550682 | -0.338731 | 26.2388 | 30.8473 |
| ETHContext60 | eth_up_strong | trend_down | short | 120m | ROBUST_EDGE_CANDIDATE | 1625 | 1418 | 9.929237 | 7.429237 | 4.929237 | 1.186445 | 48.3077 | -59.686293 | -6.768306 | 38.4651 | 24.6940 |
| CloseRelativeEMA20 | above_ema20_strong | trend_up | long | 240m | ROBUST_EDGE_CANDIDATE | 12304 | 9129 | 9.617760 | 7.117760 | 4.617760 | 1.109559 | 48.2932 | -23.749628 | -2.704220 | 25.4393 | 34.0544 |
| ETHContext60 | eth_down_strong | trend_down | long | 240m | ROBUST_EDGE_CANDIDATE | 28667 | 16397 | 9.138055 | 6.638055 | 4.138055 | 1.115614 | 54.0552 | -37.305177 | 1.799548 | 23.2241 | 18.9645 |
| BTCContext60 | btc_up_strong | compressed_up | long | 240m | ROBUST_EDGE_CANDIDATE | 12427 | 9282 | 8.458337 | 5.958337 | 3.458337 | 1.122407 | 50.3903 | -41.739451 | 3.493784 | 26.3243 | 13.3855 |
| CloseRelativeEMA20 | below_ema20_strong | trend_down | long | 240m | ROBUST_EDGE_CANDIDATE | 13207 | 9671 | 8.263914 | 5.763914 | 3.263914 | 1.085194 | 53.5928 | -42.158003 | -2.184339 | 33.8862 | 27.2631 |
| CloseRelativeEMA20 | above_ema20_strong | range | long | 120m | ROBUST_EDGE_CANDIDATE | 7404 | 6323 | 7.664527 | 5.164527 | 2.664527 | 1.105313 | 51.4722 | -41.533207 | -1.709003 | 26.5241 | 23.9228 |
| Return15 | return15_down_strong | trend_down | long | 240m | ROBUST_EDGE_CANDIDATE | 29259 | 20279 | 6.073773 | 3.573773 | 1.073773 | 1.070999 | 53.0640 | -39.529317 | -1.411047 | 35.3595 | 24.9266 |
| ETHContext60 | eth_up_strong | compressed_up | long | 240m | ROBUST_EDGE_CANDIDATE | 15140 | 11284 | 5.839557 | 3.339557 | 0.839557 | 1.086663 | 50.4557 | -24.990241 | 0.234841 | 31.7560 | 23.7266 |

## Weak Edge Buckets
| Feature | Bucket | Regime | Side | Horizon | Label | Samples | Clusters | Exp5 | Exp7.5 | Exp10 | PF | Win% | Worst Month | Worst Symbol | Top Symbol% | Top Month% |
|---|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| ATRPct14 | atr_pct_14_high | trend_up | long | 240m | WEAK_EDGE | 755 | 487 | 102.301970 | 99.801970 | 97.301970 | 1.866515 | 56.5563 | -726.153846 | -203.542870 | 26.7659 | 24.7396 |
| ATRPct14 | atr_pct_14_high | range | long | 240m | WEAK_EDGE | 1007 | 761 | 90.489491 | 87.989491 | 85.489491 | 1.642219 | 58.9871 | -184.252529 | 4.148699 | 32.4508 | 27.7768 |
| CloseRelativeEMA20 | above_ema20_strong | compressed_up | long | 240m | WEAK_EDGE | 778 | 699 | 70.848453 | 68.348453 | 65.848453 | 1.783058 | 59.1260 | -100.637685 | 41.447438 | 25.1581 | 27.0204 |
| ATRPct14 | atr_pct_14_high | trend_down | long | 240m | WEAK_EDGE | 1330 | 925 | 65.722666 | 63.222666 | 60.722666 | 1.467161 | 57.8195 | -172.565708 | 0.800844 | 38.5863 | 32.2096 |
| ATRPct14 | atr_pct_14_high | range | long | 120m | WEAK_EDGE | 1007 | 761 | 56.637184 | 54.137184 | 51.637184 | 1.500939 | 55.9086 | -169.889763 | 22.115099 | 24.0422 | 30.3726 |
| ATRPct14 | atr_pct_14_high | trend_up | long | 120m | WEAK_EDGE | 755 | 487 | 53.909150 | 51.409150 | 48.909150 | 1.460335 | 53.7748 | -511.072874 | -97.557592 | 21.9930 | 33.3315 |
| ATRPct14 | atr_pct_14_high | trend_down | long | 120m | WEAK_EDGE | 1330 | 925 | 52.095769 | 49.595769 | 47.095769 | 1.486237 | 56.9925 | -131.959151 | 2.267488 | 37.1151 | 19.1833 |
| CloseRelativeEMA20 | below_ema20_strong | compressed_down | long | 240m | WEAK_EDGE | 686 | 645 | 50.422200 | 47.922200 | 45.422200 | 1.469071 | 56.2682 | -101.404627 | -14.007804 | 27.2442 | 21.3367 |
| ATRPct14 | atr_pct_14_high | trend_down | long | 60m | WEAK_EDGE | 1330 | 925 | 47.635339 | 45.135339 | 42.635339 | 1.608706 | 59.6992 | -100.088423 | 21.431486 | 31.4643 | 21.5693 |
| CloseRelativeEMA20 | below_ema20_strong | compressed_down | long | 120m | WEAK_EDGE | 686 | 645 | 41.942095 | 39.442095 | 36.942095 | 1.516073 | 56.2682 | -68.292101 | 2.759658 | 33.9337 | 19.1988 |
| CloseRelativeEMA20 | above_ema20_strong | compressed_up | long | 120m | WEAK_EDGE | 778 | 699 | 36.249935 | 33.749935 | 31.249935 | 1.481279 | 57.9692 | -64.239701 | 3.789977 | 26.2187 | 32.0666 |
| Return15 | return15_down_strong | compressed_down | long | 240m | WEAK_EDGE | 1168 | 1104 | 34.881080 | 32.381080 | 29.881080 | 1.341301 | 55.9932 | -114.244921 | -16.317580 | 29.0246 | 33.1011 |
| Return15 | return15_down_strong | compressed_down | long | 120m | WEAK_EDGE | 1168 | 1104 | 33.935062 | 31.435062 | 28.935062 | 1.458399 | 55.7363 | -95.338183 | -18.229097 | 24.7734 | 24.9474 |
| CloseRelativeEMA20 | above_ema20_strong | compressed_up | long | 60m | WEAK_EDGE | 778 | 699 | 31.401513 | 28.901513 | 26.401513 | 1.610475 | 55.5270 | -56.496278 | 12.380044 | 25.7862 | 27.9792 |
| Return15 | return15_up_strong | compressed_down | long | 240m | WEAK_EDGE | 918 | 888 | 30.092968 | 27.592968 | 25.092968 | 1.296546 | 53.1590 | -101.664180 | 2.473445 | 28.5618 | 31.4742 |
| Return15 | return15_down_strong | compressed_up | long | 240m | WEAK_EDGE | 823 | 788 | 24.460731 | 21.960731 | 19.460731 | 1.247688 | 51.2758 | -131.507173 | -7.698478 | 35.6211 | 28.0980 |
| ATRPct14 | atr_pct_14_high | range | long | 60m | WEAK_EDGE | 1007 | 761 | 21.615735 | 19.115735 | 16.615735 | 1.213616 | 52.7309 | -124.628741 | -14.414024 | 23.9287 | 32.1394 |
| Return15 | return15_up_strong | compressed_up | long | 240m | WEAK_EDGE | 1271 | 1178 | 18.873414 | 16.373414 | 13.873414 | 1.180406 | 51.5342 | -91.612480 | -6.588443 | 34.6084 | 32.3851 |
| CloseRelativeEMA20 | below_ema20_strong | compressed_down | long | 60m | WEAK_EDGE | 686 | 645 | 17.932610 | 15.432610 | 12.932610 | 1.243876 | 55.6851 | -116.761035 | -12.411869 | 25.4096 | 18.8875 |
| ATRPct14 | atr_pct_14_high | trend_down | long | 30m | WEAK_EDGE | 1330 | 925 | 17.567377 | 15.067377 | 12.567377 | 1.224376 | 58.8722 | -86.637815 | -1.868739 | 42.9956 | 24.9328 |
| Return15 | return15_down_strong | compressed_up | long | 120m | WEAK_EDGE | 823 | 788 | 17.087125 | 14.587125 | 12.087125 | 1.237664 | 52.9769 | -113.108571 | -2.379345 | 34.8606 | 25.7725 |
| CloseRelativeEMA20 | above_ema20_strong | compressed_up | long | 30m | WEAK_EDGE | 778 | 699 | 14.763145 | 12.263145 | 9.763145 | 1.323832 | 50.8997 | -63.352611 | 0.839250 | 41.5161 | 31.5178 |
| Return15 | return15_down_strong | compressed_down | long | 60m | WEAK_EDGE | 1168 | 1104 | 14.275063 | 11.775063 | 9.275063 | 1.230799 | 54.1952 | -77.759709 | -15.540516 | 22.7852 | 26.4052 |
| Return15 | return15_up_strong | compressed_up | long | 60m | WEAK_EDGE | 1271 | 1178 | 13.963986 | 11.463986 | 8.963986 | 1.267789 | 52.4784 | -50.116878 | 9.704384 | 17.8447 | 33.0870 |
| Return15 | return15_up_strong | compressed_down | long | 120m | WEAK_EDGE | 918 | 888 | 13.382172 | 10.882172 | 8.382172 | 1.173819 | 50.9804 | -72.431310 | -18.015506 | 30.3248 | 28.5720 |
| Return15 | return15_down_strong | compressed_up | long | 60m | WEAK_EDGE | 823 | 788 | 13.266678 | 10.766678 | 8.266678 | 1.240210 | 52.9769 | -69.789789 | -7.032591 | 28.4289 | 29.5569 |
| CloseRelativeEMA20 | below_ema20_strong | compressed_down | long | 30m | WEAK_EDGE | 686 | 645 | 12.376093 | 9.876093 | 7.376093 | 1.230061 | 54.3732 | -92.968278 | -13.024259 | 31.8595 | 23.6864 |
| Return15 | return15_down_strong | compressed_up | long | 30m | WEAK_EDGE | 823 | 788 | 10.467128 | 7.967128 | 5.467128 | 1.266285 | 51.0328 | -100.870119 | -1.776538 | 31.2647 | 26.7568 |
| BTCContext60 | btc_down_strong | trend_up | long | 120m | WEAK_EDGE | 928 | 814 | 7.940070 | 5.440070 | 2.940070 | 1.118450 | 48.1681 | -70.674560 | -23.243056 | 42.0912 | 28.7570 |
| ATRPct14 | atr_pct_14_high | trend_up | long | 15m | WEAK_EDGE | 755 | 487 | 7.759558 | 5.259558 | 2.759558 | 1.119633 | 48.0795 | -166.767264 | -8.383534 | 35.7929 | 23.2374 |
| CloseRelativeEMA20 | below_ema20_strong | compressed_down | long | 15m | WEAK_EDGE | 686 | 645 | 7.757659 | 5.257659 | 2.757659 | 1.212142 | 56.5598 | -46.692559 | -12.352502 | 26.9133 | 23.8271 |
| Return15 | return15_down_strong | compressed_down | long | 30m | WEAK_EDGE | 1168 | 1104 | 5.542746 | 3.042746 | 0.542746 | 1.117095 | 52.3973 | -58.778700 | -30.062211 | 25.6468 | 26.4977 |
| SessionUTC | us | compressed_down | short | 240m | WEAK_EDGE | 13459 | 9794 | 5.004446 | 2.504446 | 0.004446 | 1.069472 | 48.0125 | -63.227472 | -4.333733 | 34.5549 | 26.1555 |
| SessionUTC | europe | trend_up | short | 240m | WEAK_EDGE | 30756 | 15080 | 4.826132 | 2.326132 | -0.173868 | 1.082211 | 52.1037 | -32.544559 | -9.039371 | 30.8418 | 19.1474 |
| RealizedVol60 | realized_vol_60_mid | trend_down | long | 120m | WEAK_EDGE | 18013 | 9618 | 4.756257 | 2.256257 | -0.243743 | 1.061359 | 53.4836 | -39.136152 | -6.642758 | 29.9240 | 26.8458 |
| CloseRelativeEMA20 | above_ema20 | trend_down | long | 60m | WEAK_EDGE | 2094 | 2021 | 4.484620 | 1.984620 | -0.515380 | 1.104803 | 53.1519 | -59.793459 | -9.976685 | 25.5571 | 34.3153 |
| Return15 | return15_up | trend_down | long | 240m | WEAK_EDGE | 2136 | 2111 | 4.250599 | 1.750599 | -0.749401 | 1.059614 | 51.5918 | -27.016667 | -17.415180 | 32.1302 | 24.9971 |
| SessionUTC | late_us | compressed_down | long | 120m | WEAK_EDGE | 6263 | 4772 | 4.248359 | 1.748359 | -0.751641 | 1.096498 | 50.5030 | -44.903775 | -7.785674 | 42.9611 | 19.9832 |
| BTCContext60 | btc_up_strong | range | long | 240m | WEAK_EDGE | 10002 | 7967 | 3.984027 | 1.484027 | -1.015973 | 1.052802 | 50.4699 | -45.758747 | -2.512677 | 40.1359 | 24.1127 |
| SessionUTC | asia | trend_up | long | 240m | WEAK_EDGE | 36588 | 17818 | 3.958634 | 1.458634 | -1.041366 | 1.079105 | 49.1199 | -25.551612 | 0.526113 | 21.5179 | 33.4294 |
| BTCContext60 | btc_down_strong | trend_down | long | 240m | WEAK_EDGE | 19720 | 11580 | 3.807961 | 1.307961 | -1.192039 | 1.045162 | 53.8235 | -42.990897 | -13.210058 | 43.3857 | 22.9729 |
| SessionUTC | asia | range | long | 240m | WEAK_EDGE | 70306 | 31991 | 3.714083 | 1.214083 | -1.285917 | 1.072381 | 50.0583 | -16.357673 | -0.294350 | 24.0210 | 30.2722 |
| CloseRelativeEMA20 | above_ema20_strong | trend_up | long | 120m | WEAK_EDGE | 12304 | 9129 | 3.701589 | 1.201589 | -1.298411 | 1.054393 | 47.0741 | -26.430876 | -2.643399 | 23.8986 | 26.2735 |
| Return15 | return15_up_strong | range | long | 240m | WEAK_EDGE | 19019 | 15548 | 3.548163 | 1.048163 | -1.451837 | 1.042766 | 51.7693 | -54.920483 | -4.727254 | 42.2679 | 24.3872 |
| ETHContext60 | eth_up_strong | trend_down | short | 240m | WEAK_EDGE | 1625 | 1418 | 3.493619 | 0.993619 | -1.506381 | 1.045256 | 47.8769 | -80.680216 | -18.164152 | 41.3979 | 34.8626 |
| BTCContext60 | btc_up_strong | trend_down | long | 240m | WEAK_EDGE | 759 | 705 | 3.467112 | 0.967112 | -1.532888 | 1.042951 | 51.3834 | -105.499058 | -15.943787 | 34.1136 | 28.7579 |
| SessionUTC | asia | compressed_up | long | 240m | WEAK_EDGE | 13062 | 9148 | 3.411283 | 0.911283 | -1.588717 | 1.059766 | 50.7809 | -36.618936 | -2.314767 | 21.0862 | 16.7700 |
| ATRPct14 | atr_pct_14_mid | compressed_up | long | 240m | WEAK_EDGE | 15927 | 11173 | 3.077089 | 0.577089 | -1.922911 | 1.035885 | 50.1224 | -48.321154 | -7.988667 | 26.5832 | 31.8393 |
| CloseRelativeEMA20 | below_ema20_strong | range | long | 120m | WEAK_EDGE | 6205 | 5393 | 3.068902 | 0.568902 | -1.931098 | 1.043267 | 50.9589 | -40.258695 | -6.606726 | 27.5735 | 34.5928 |
| ETHContext60 | eth_down_strong | range | long | 240m | WEAK_EDGE | 16679 | 12476 | 2.789712 | 0.289712 | -2.210288 | 1.035636 | 51.2321 | -46.726362 | -2.321753 | 34.0282 | 27.1477 |
| BTCContext60 | btc_up_strong | trend_up | long | 240m | WEAK_EDGE | 17436 | 10827 | 2.762758 | 0.262758 | -2.237242 | 1.039880 | 48.4916 | -19.394458 | -1.325633 | 23.6534 | 23.3663 |
| BTCContext60 | btc_down_strong | range | long | 120m | WEAK_EDGE | 9567 | 7600 | 2.727892 | 0.227892 | -2.272108 | 1.044240 | 51.4372 | -48.141700 | -4.524685 | 29.7805 | 25.7507 |
| ETHContext60 | eth_up_strong | range | long | 240m | WEAK_EDGE | 17543 | 13106 | 2.708880 | 0.208880 | -2.291120 | 1.039007 | 49.5810 | -25.666872 | -6.257383 | 29.8498 | 28.7105 |
| SessionUTC | asia | trend_down | long | 240m | WEAK_EDGE | 33470 | 16334 | 2.625731 | 0.125731 | -2.374269 | 1.045719 | 51.2608 | -35.618659 | -3.405078 | 38.5816 | 17.9075 |
| VolumeRatio20 | volume_ratio_low | compressed_up | long | 240m | WEAK_EDGE | 18592 | 15813 | 2.548139 | 0.048139 | -2.451861 | 1.041018 | 49.6880 | -21.688943 | -1.879261 | 23.0266 | 31.4033 |
| SessionUTC | asia | compressed_down | long | 240m | WEAK_EDGE | 10318 | 7817 | 2.443227 | -0.056773 | -2.556773 | 1.039798 | 49.4960 | -68.985748 | -1.503909 | 36.1704 | 24.2465 |
| BTCContext60 | btc_up_strong | range | long | 120m | WEAK_EDGE | 10002 | 7967 | 1.969783 | -0.530217 | -3.030217 | 1.036278 | 49.4001 | -30.397873 | -2.187732 | 32.5269 | 24.0975 |
| CloseRelativeEMA20 | below_ema20_strong | trend_down | long | 120m | WEAK_EDGE | 13207 | 9671 | 1.887403 | -0.612597 | -3.112597 | 1.025003 | 53.8502 | -38.834945 | -3.243333 | 35.4868 | 30.7198 |
| ETHContext60 | eth_down | compressed_down | short | 240m | WEAK_EDGE | 14985 | 12760 | 1.794482 | -0.705518 | -3.205518 | 1.028925 | 48.6820 | -37.196893 | -4.676083 | 42.8499 | 33.3375 |
| SessionUTC | late_us | compressed_up | long | 240m | WEAK_EDGE | 8395 | 5804 | 1.791427 | -0.708573 | -3.208573 | 1.029958 | 50.3633 | -80.830958 | -19.051940 | 36.3405 | 31.4508 |
| BTCContext60 | btc_down_strong | trend_up | long | 15m | WEAK_EDGE | 928 | 814 | 1.764681 | -0.735319 | -3.235319 | 1.065534 | 50.1078 | -32.253669 | -8.855847 | 44.3620 | 24.0772 |
| ATRPct14 | atr_pct_14_mid | trend_down | long | 240m | WEAK_EDGE | 44654 | 24466 | 1.757906 | -0.742094 | -3.242094 | 1.021152 | 52.7254 | -33.555398 | -6.763837 | 43.6224 | 32.0036 |
| RealizedVol60 | realized_vol_60_low | compressed_down | short | 240m | WEAK_EDGE | 28359 | 21606 | 1.689873 | -0.810127 | -3.310127 | 1.029999 | 48.5948 | -41.039581 | -5.709496 | 32.1006 | 23.5212 |
| BTCContext60 | btc_down_strong | trend_up | long | 60m | WEAK_EDGE | 928 | 814 | 1.524005 | -0.975995 | -3.475995 | 1.029600 | 46.2284 | -60.020513 | -19.255999 | 44.6288 | 24.9600 |
| BTCContext60 | btc_down_strong | range | long | 240m | WEAK_EDGE | 9567 | 7600 | 0.694398 | -1.805602 | -4.305602 | 1.008355 | 51.3954 | -72.498461 | -11.657183 | 28.5195 | 30.9562 |
| SessionUTC | europe | range | short | 240m | WEAK_EDGE | 61231 | 27836 | 0.584769 | -1.915231 | -4.415231 | 1.009975 | 50.4891 | -38.978262 | -9.713826 | 35.9174 | 27.6077 |
| BTCContext60 | btc_up_strong | compressed_up | long | 120m | WEAK_EDGE | 12427 | 9282 | 0.561348 | -1.938652 | -4.438652 | 1.010663 | 48.5556 | -23.596402 | -6.982588 | 41.9897 | 19.7165 |
| Return15 | return15_down | trend_up | long | 240m | WEAK_EDGE | 2286 | 2245 | 0.511281 | -1.988719 | -4.488719 | 1.007354 | 49.3001 | -66.644663 | -18.018063 | 43.1104 | 21.9691 |
| ETHContext60 | eth_up_strong | trend_up | long | 240m | WEAK_EDGE | 25344 | 15253 | 0.446861 | -2.053139 | -4.553139 | 1.006608 | 47.9009 | -25.599500 | -2.832950 | 33.1003 | 29.9211 |
| TakerBuyRatio | taker_balanced | trend_down | long | 240m | WEAK_EDGE | 15801 | 13594 | 0.110466 | -2.389534 | -4.889534 | 1.001563 | 51.1170 | -52.795971 | -9.766256 | 37.6606 | 27.7569 |
| SessionUTC | asia | compressed_up | long | 120m | WEAK_EDGE | 13062 | 9148 | 0.031313 | -2.468687 | -4.968687 | 1.000763 | 49.6555 | -42.100146 | -3.953041 | 44.4950 | 23.2705 |

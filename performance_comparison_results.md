# Performance Comparison: NestJS vs Go (Fiber)

Generated on: 2026-07-13T23:22:59.103Z
Total Benchmark Suite Duration: **276.2 seconds**
NestJS Run Duration: **137.8s** | Go Run Duration: **133.9s**
Concurrency: 50 connections, Duration: 10s per endpoint.

## [General] Root Endpoint (`/`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 15628.37 | 150184.73 | **9.61x** requests/sec |
| **Latency (Avg)** | 2.54 ms | 0.01 ms | **99.6%** faster |
| **Latency (P50)** | 2.00 ms | 0.00 ms | - |
| **Latency (P99)** | 6.00 ms | 0.00 ms | - |
| **Errors** | 0 | 0 | - |

## [General] System Config (`/api/v1/config`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 18168.00 | 149661.10 | **8.24x** requests/sec |
| **Latency (Avg)** | 2.24 ms | 0.01 ms | **99.6%** faster |
| **Latency (P50)** | 2.00 ms | 0.00 ms | - |
| **Latency (P99)** | 5.00 ms | 0.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Auth] Login API (`/api/v1/auth/login`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 25.30 | 45.30 | **1.79x** requests/sec |
| **Latency (Avg)** | 1781.28 ms | 1044.63 ms | **41.4%** faster |
| **Latency (P50)** | 1805.00 ms | 1237.00 ms | - |
| **Latency (P99)** | 2478.00 ms | 1419.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Products] Get All Products (`/api/v1/products`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 35.80 | 220.00 | **6.15x** requests/sec |
| **Latency (Avg)** | 1311.24 ms | 223.77 ms | **82.9%** faster |
| **Latency (P50)** | 1093.00 ms | 180.00 ms | - |
| **Latency (P99)** | 3100.00 ms | 913.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Products] Get Seasonal Products (`/api/v1/products/seasonal`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 43.90 | 226.90 | **5.17x** requests/sec |
| **Latency (Avg)** | 1080.10 ms | 217.67 ms | **79.8%** faster |
| **Latency (P50)** | 981.00 ms | 180.00 ms | - |
| **Latency (P99)** | 2217.00 ms | 924.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Products] Get Top Rated Products (`/api/v1/products/top-rated`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 22.30 | 213.40 | **9.57x** requests/sec |
| **Latency (Avg)** | 2005.70 ms | 231.46 ms | **88.5%** faster |
| **Latency (P50)** | 1968.00 ms | 179.00 ms | - |
| **Latency (P99)** | 3036.00 ms | 926.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Products] Get Product Categories (`/api/v1/products/categories`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 45.50 | 223.30 | **4.91x** requests/sec |
| **Latency (Avg)** | 1051.76 ms | 221.63 ms | **78.9%** faster |
| **Latency (P50)** | 907.00 ms | 179.00 ms | - |
| **Latency (P99)** | 2625.00 ms | 929.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Locations] Get Locations (`/api/v1/locations`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 50.00 | 224.20 | **4.48x** requests/sec |
| **Latency (Avg)** | 956.14 ms | 220.12 ms | **77.0%** faster |
| **Latency (P50)** | 910.00 ms | 180.00 ms | - |
| **Latency (P99)** | 1743.00 ms | 930.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Stars] Get Perks (`/api/v1/perks`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 16760.37 | 155490.91 | **9.28x** requests/sec |
| **Latency (Avg)** | 2.35 ms | 0.01 ms | **99.6%** faster |
| **Latency (P50)** | 2.00 ms | 0.00 ms | - |
| **Latency (P99)** | 6.00 ms | 0.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Stars] Get Stars Balance (`/api/v1/stars/balance`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 54.10 | 220.60 | **4.08x** requests/sec |
| **Latency (Avg)** | 878.71 ms | 224.16 ms | **74.5%** faster |
| **Latency (P50)** | 909.00 ms | 179.00 ms | - |
| **Latency (P99)** | 953.00 ms | 918.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Stars] Get Rewards List (`/api/v1/stars/rewards`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 50.20 | 206.60 | **4.12x** requests/sec |
| **Latency (Avg)** | 950.07 ms | 238.89 ms | **74.9%** faster |
| **Latency (P50)** | 911.00 ms | 181.00 ms | - |
| **Latency (P99)** | 1669.00 ms | 927.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Users] Get User Profile (`/api/v1/users/profile`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 25.00 | 78.70 | **3.15x** requests/sec |
| **Latency (Avg)** | 1836.67 ms | 617.25 ms | **66.4%** faster |
| **Latency (P50)** | 1825.00 ms | 541.00 ms | - |
| **Latency (P99)** | 2674.00 ms | 1298.00 ms | - |
| **Errors** | 0 | 0 | - |

## [Dashboard] Get Dashboard Summary (`/api/v1/dashboard/summary`)

| Metric | NestJS | Go (Fiber) | Change / Improvement |
| --- | --- | --- | --- |
| **Req/sec (Avg)** | 5.10 | 46.90 | **9.20x** requests/sec |
| **Latency (Avg)** | 5882.93 ms | 1026.44 ms | **82.6%** faster |
| **Latency (P50)** | 5892.00 ms | 901.00 ms | - |
| **Latency (P99)** | 6470.00 ms | 2370.00 ms | - |
| **Errors** | 0 | 0 | - |


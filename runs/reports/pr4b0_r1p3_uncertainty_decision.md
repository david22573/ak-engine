# PR4B0-R1P3 Uncertainty Decision

Decision: **REVISE V1; ACCEPT V2**. V1 remains proposed and unaccepted.

Accepted version: `ak.engine.uncertainty.cluster-bootstrap.v2`

Accepted hash: `sha256:1a91541c94378cc6f34e62a39ae504d3d013b5dab63a2b622641cdd1088148fb`

The accepted method estimates mean mandatory-cost net expectancy per independent cluster. It uses exactly 10,000 unstratified nonparametric cluster-bootstrap replicates, N draws with replacement per replicate, ascending numeric sort, and the nearest-rank fifth-percentile element at zero-based index 499. Qualification requires N >= 300 and lower bound > 0. N < 30 is not reportable; 30 <= N < 300 is reportable but fails the sample gate. The seed binds every required canonical identity.

Acceptance was decided using synthetic fixtures only and without inspecting candidate-performance results.

Decision-record hash: `sha256:a321aea5beec64394466a539e9a0bbf454df3f7e2fc23b27b414829e49d0c670`
Artifact hash: `sha256:634f6ccd30cac3f1cb0912cfac17f1df7956476174306c0853373fa1935b101d`

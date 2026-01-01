# Multi-Service Music Search Architecture

## Overview
Refactor the music search system to support multiple music sources with intelligent ranking and deduplication.

---

## Architecture Design

### 1. **Core Components**

#### A. Music Provider Interface
Abstract interface that any music source must implement:
```go
type MusicProvider interface {
    Name() string
    Search(ctx context.Context, query string) ([]ProviderResult, error)
    Priority() int  // Higher = more trusted source
}
```

#### B. Provider Result
Extended result with metadata for ranking:
```go
type ProviderResult struct {
    Song          domain.Song
    Provider      string
    MatchScore    float64  // How well it matches the query
    ProviderRank  int      // Position in provider's results (1-indexed)
}
```

#### C. Aggregator Service
Orchestrates multiple providers and merges results:
```go
type MusicAggregatorService struct {
    providers []MusicProvider
    ranker    *ResultRanker
}
```

#### D. Result Ranker
Implements smart ranking algorithm:
```go
type ResultRanker struct {
    // Configurable weights
    providerPriorityWeight float64
    matchScoreWeight       float64
    positionWeight         float64
    diversityBonus         float64
}
```

---

## 2. **Ranking Algorithm**

### Scoring Factors

**Final Score = (ProviderScore × 0.3) + (MatchScore × 0.4) + (PositionScore × 0.2) + (DiversityBonus × 0.1)**

#### A. Provider Priority Score (30% weight)
- Each provider has a priority rating (1-10)
- More reliable/accurate sources get higher priority
- Normalized to 0-1 range

#### B. Match Score (40% weight - MOST IMPORTANT)
Relevance to search query based on:
- **Exact title match**: 1.0
- **Title contains all query words**: 0.9
- **Title contains some query words**: 0.7
- **Artist matches query**: 0.8
- **Title + Artist combined match**: Dynamic scoring
- **Fuzzy matching**: Levenshtein distance for typos

#### C. Position Score (20% weight)
- Results appearing earlier in provider results score higher
- Formula: `1.0 / (1.0 + log(position))`
- First result: ~1.0, 10th result: ~0.3, 100th result: ~0.2

#### D. Diversity Bonus (10% weight)
- Boost songs from underrepresented artists
- Prevents single artist dominating results
- Applied after initial scoring

### Deduplication Strategy

**Matching Criteria** (Songs are considered duplicates if):
1. Exact title + artist match (case-insensitive)
2. Normalized title match (remove special chars, extra spaces)
3. High similarity score (>85% Levenshtein similarity)

**Merge Logic:**
- Keep the result with highest combined score
- Prefer result with better metadata (image, link quality)
- Preserve provider attribution

---

## 3. **Implementation Plan**

### Phase 1: Create Abstractions
1. Define `MusicProvider` interface
2. Create `ProviderResult` domain model
3. Create `ResultRanker` utility
4. Create `MusicAggregatorService`

### Phase 2: Refactor Existing Service
1. Convert `MusicFinderService` → `MuzVibeProvider`
2. Implement `MusicProvider` interface
3. Add match scoring logic

### Phase 3: Implement Ranking
1. Build relevance scoring algorithm
2. Implement deduplication logic
3. Add configurable weights
4. Create result merger

### Phase 4: Integration
1. Wire up aggregator in DI container
2. Update handler to use aggregator
3. Keep backward compatibility

### Phase 5: Future Extensibility
1. Easy to add new providers (Spotify API, YouTube API, etc.)
2. Provider-specific configuration
3. A/B testing different ranking weights

---

## 4. **Code Structure**

```
internal/
├── core/
│   ├── domain/
│   │   ├── song.go                    # Existing
│   │   ├── provider_result.go         # NEW - Provider result with metadata
│   │   └── search_config.go           # NEW - Search configuration
│   ├── ports/
│   │   ├── music_finder_ports.go      # Update interface
│   │   └── music_provider_port.go     # NEW - Provider interface
│   └── services/
│       ├── music_aggregator_service.go    # NEW - Multi-provider orchestrator
│       ├── result_ranker.go               # NEW - Ranking algorithm
│       └── providers/
│           ├── muzvibe_provider.go        # NEW - Refactored from music_finder_service
│           └── (future: spotify_provider.go, youtube_provider.go)
```

---

## 5. **Benefits**

### Immediate
✅ Better search relevance (smart ranking vs random order)  
✅ Reduced irrelevant results  
✅ Cleaner architecture (separation of concerns)

### Future
✅ Easy to add new music sources  
✅ Can compare provider quality  
✅ A/B test ranking algorithms  
✅ Provider fallback/redundancy  
✅ Parallel provider queries for speed

---

## 6. **Example Flow**

```
User searches: "pizza tower ost"

1. Aggregator queries all providers in parallel:
   - MuzVibeProvider returns 20 results
   - (Future: YouTubeProvider returns 15 results)

2. Each result gets scored:
   Result A: "PIZZA TOWER OST" by "Artist A" (position 1, muzvibe)
     - Provider: 8/10 = 0.8
     - Match: 1.0 (exact match)
     - Position: 1.0 (first)
     - Score: (0.8×0.3) + (1.0×0.4) + (1.0×0.2) = 0.84

   Result B: "Pizza song compilation" by "Artist B" (position 2, muzvibe)
     - Provider: 8/10 = 0.8
     - Match: 0.5 (partial match)
     - Position: 0.83
     - Score: (0.8×0.3) + (0.5×0.4) + (0.83×0.2) = 0.61

3. Deduplicate (merge identical songs from different providers)

4. Sort by final score

5. Return top N results
```

---

## 7. **Configuration**

Allow runtime configuration:
```go
type SearchConfig struct {
    MaxResults              int     // Default: 20
    EnableParallelSearch    bool    // Default: true
    ProviderTimeout         time.Duration
    RankingWeights          RankingWeights
    EnableDeduplication     bool    // Default: true
    SimilarityThreshold     float64 // For dedup
}
```

---

## Next Steps

1. Implement provider abstraction layer
2. Refactor existing muzvibe service to provider pattern
3. Build ranking algorithm with configurable weights
4. Add comprehensive tests for ranking logic
5. Update API handler to use new aggregator

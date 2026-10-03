// Mirrors backend/internal/search/service.go's Result struct and the
// searchResponse struct in backend/internal/api/search_handler.go.
// Keep these in sync by hand for now; codegen from the Go types is a
// nice Phase 4 upgrade once the API shape stabilizes.

export interface MemberResult {
  id: string;
  full_name: string;
  headline?: string;
  bio?: string;
  location?: string;
  rank: number;
}

export interface SearchResponse {
  query: string;
  results: MemberResult[];
}

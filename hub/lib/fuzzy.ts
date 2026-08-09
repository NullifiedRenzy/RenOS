/**
 * Lightweight fuzzy matcher for the command center.
 *
 * Scores a query against a target string using a subsequence match
 * (like VS Code / Sublime). Returns a score >= 0 when matched, or -1 when
 * the query is not a subsequence. Bonus points are awarded for consecutive
 * characters, matches at word starts, and prefix matches.
 */
export interface FuzzyMatch {
  score: number;
  matched: boolean;
}

export function fuzzyMatch(query: string, target: string): FuzzyMatch {
  if (query.length === 0) {
    return { score: 0, matched: false };
  }

  const q = query.toLowerCase();
  const t = target.toLowerCase();

  if (q.length > t.length) {
    return { score: -1, matched: false };
  }

  let qi = 0;
  let score = 0;
  let prevIndex = -2;
  let consecutive = 0;

  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) {
      // Bonus for consecutive characters.
      if (ti === prevIndex + 1) {
        consecutive += 1;
        score += 2 * consecutive;
      } else {
        consecutive = 1;
        score += 1;
      }
      // Bonus for matches at the start of a word.
      if (ti === 0 || t[ti - 1] === " " || t[ti - 1] === "-" || t[ti - 1] === "/") {
        score += 5;
      }
      // Bonus for exact prefix.
      if (ti === 0 && qi === 0) {
        score += 3;
      }
      // Slight penalty for gaps (typo tolerance is rewarded, huge gaps are not).
      if (ti - prevIndex > 3 && prevIndex >= 0) {
        score -= 1;
      }
      prevIndex = ti;
      qi++;
    }
  }

  if (qi < q.length) {
    return { score: -1, matched: false };
  }

  // Bonus for short queries that are exact substring/prefix.
  if (t.startsWith(q)) score += 8;
  if (t === q) score += 12;

  return { score, matched: true };
}

/** Returns entries whose fuzzy score exceeds a threshold, sorted best-first. */
export function fuzzyRank<T>(
  items: T[],
  query: string,
  getText: (item: T) => string,
  limit = 8,
): T[] {
  if (!query.trim()) {
    return items.slice(0, limit);
  }
  const scored: Array<{ item: T; score: number }> = [];
  for (const item of items) {
    const { score, matched } = fuzzyMatch(query, getText(item));
    if (matched && score > 0) {
      scored.push({ item, score });
    }
  }
  scored.sort((a, b) => b.score - a.score);
  return scored.slice(0, limit).map((s) => s.item);
}

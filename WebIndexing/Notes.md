What Web Indexing Actually Is ?
At its core, web indexing is just like the index at the back of a textbook.
<br>
The Problem: If you have billions of web pages, you cannot scan every page line-by-line to see if it contains the word "pizza" whenever a user searches for it.
That would take forever.
<br>
The Solution: You build an Inverted Index. Instead of mapping URL ➔ Content, you turn it inside out to map Word ➔ List of URLs containing that word.

The Sub-Questions You Must Ask Next
<br>
Once you align on the basic definition, you immediately need to lock down the scale and constraints to design the database. You should follow up by asking:

- What is the scale of data? Are we indexing 100 million pages or the entire public web (e.g., 50+ billion pages)?
- What are we storing per page? Just the URL and word counts, or also snippets, titles, and PageRank scores?
- What is the Read-to-Write ratio? Is the database heavily slammed by user search queries, or are we designing this purely for background batch-writes from a web crawler?
- What are the search requirements? Do we need to support exact keyword matches, wildcards, or full complex boolean phrases (e.g., "pizza" AND "pepperoni")?

```
Why would you ask, what are we storing per page ?
What is the reasoning behind this ?

How did we reach to asking read-to-write ratio when talking about web indexing ? how is background batch-writes different from a web crawler ?

What is the difference between keyword matches, wildcards, and full complex boolean phrases explain ?
```

Why Ask What We Store Per Page?
<br>
Asking what we store per page determines the storage capacity and database schema.If you only store a Word and a list of Document IDs, the data size is very small. If you must also store word positions (for phrase matches), text snippets, page titles, and PageRank scores, the data size multiplies.Knowing the exact payload allows you to calculate the total petabytes required. This tells you if the index can fit in RAM for speed or if it must live on disk.

Understanding the Read-to-Write Ratio
<br>
Every database design changes radically based on whether it is read-heavy or write-heavy.
<br>
High Read Ratio: If the index is queried by users billions of times a second, you must prioritize massive caching, replication, and ultra-low read latency.
<br>
High Write Ratio: If the system is mostly ingestion, you must prioritize fast data appending and bulk-loading strategies.

Batch-Writes vs. Web Crawlers
<br>
A web crawler is a bot that continuously browses the internet to download raw HTML pages. It does not index them; it just fetches them.
<br>
Background batch-writes happen after the crawler finishes. A separate system (like a MapReduce job) takes thousands of downloaded pages, parses them, extracts the words, and writes them into the index database all at once in large batches.The crawler finds the data, while the batch-writer formats and saves it into your index.

Types of Search Queries Explained
<br>
The database structure changes based on how users are allowed to search.
<br>
[User Query]

- ───► Keyword Match (Simple lookup)
- ───► Wildcard Search (Requires suffix trees)
- ───► Boolean Phrase (Requires merging pointer lists)

1. Keyword Matches <br>
   - What it is: Searching for a single, exact word.
   - Example: Searching for pizza.
   - Database impact: Extremely simple. The database looks up the key "pizza" in the inverted index and returns the attached list of URLs.

2. Wildcards

- What it is: Searching for a partial word using a placeholder (like \*).
- Example: Searching for comput\* to find "computer", "computing", or "computation".
- Database impact: Harder. A standard inverted index cannot look up partial words easily. You need extra data structures like a Trie or a Suffix Tree to find all words starting with comput before pulling their URLs.

3. Full Complex Boolean Phrases

- What it is: Combining multiple terms using logic operators like AND, OR, or NOT.
- Example: Searching for "pizza" AND "pepperoni" NOT "anchovies".
- Database impact: Complex. The database must look up the list for "pizza", look up the list for "pepperoni", and perform a mathematical intersection (finding URLs present in both lists) while filtering out the "anchovies" list. The database needs fast bitset operations or sorted pointer merging to do this instantly.

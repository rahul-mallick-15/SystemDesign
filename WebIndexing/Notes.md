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

```
"This tells you if the index can fit in RAM for speed or if it must live on disk."
"Index Database"
What is the meaning of "index" here ?
Say for simplicity, we want to do the web indexing based on words in the web page, what will be the input and output ?
```

The Meaning of "Index" Here
<br>
In this specific context, "index" refers to the database itself—specifically, the Inverted Index data structure.When people say "the index fits in RAM," they mean the entire dictionary of words and their corresponding lists of web page links is small enough to load into a computer's fast working memory. If it is too massive, it must be saved onto slower, long-term storage drives (disks).

Input and Output for Word-Based Web Indexing
<br>
If we strip away all complexity and focus strictly on indexing words, the process looks like a pipeline with a clear input and a clear output.
<br>
[INPUT] Raw HTML Pages ───► [INDEXING SYSTEM] ───► [OUTPUT] Inverted Index Database

1. The Input
   The input is a massive collection of unstructured, raw web pages fetched from the internet.Format: Raw HTML text files.
   Example Input Data:

- Page 1 (://pizza.com): "We make the best hot pizza."
- Page 2 (://cooking.com): "Learn the art of cooking pizza."

2. The Processing (What the system does)
   The indexing system cleans the input. It strips out HTML tags, removes common filler words (like "we", "the", "of"), converts everything to lowercase, and splits the sentences into individual words.

3. The Output
   The output is a structured database (the Inverted Index) that maps each unique word to a list of IDs or URLs where that word can be found.
   Format: A Key-Value lookup table.

   Example Output Data:
   <br>
   Key (Word) Value (List of URLs)
   - "make"[://pizza.com]
   - "best"[://pizza.com]
   - "hot"[://pizza.com]
   - "pizza"[://pizza.com, ://cooking.com]
   - "learn"[://cooking.com]
   - "art"[://cooking.com]
   - "cooking"[://cooking.com]

```
"Scale and Storage (The "Fit in RAM" problem)"
As of now we don't even know how much data will come from one page, is it possible to think about sharding of the inverted index (word -> url) ?
One url could be contains 1000s of words, meaning with 2 urls we could end up at  1500 indexes ?
```

You cannot design the sharding strategy until you calculate the size of the data coming from those pages. This step is called Back-of-the-Envelope Estimation. Before building anything, you must pause and do the math to see exactly how big your inverted index will become.

If Website A has 1,000 words and Website B has 1,000 words, many of those words will overlap (like "the", "is", "pizza").Instead of 2,000 separate rows in our database, the overlapping words collapse into single rows with a list of URLs.

- Unique Words (Vocabulary): This grows slowly because human languages have a limited number of words.
- The URL Lists (Posting Lists): This list grows longer and longer with every new website we index.

Doing the Math: How Big is Google's Index?
<br>
Let’s estimate the storage required for a Google-scale Inverted Index using standard interview assumptions.

1. The Raw Numbers

- Number of Web Pages: Let's assume Google indexes 100 Billion pages.
- Unique Words in Vocabulary: Let's assume there are 10 Million unique words (including names, numbers, and codes).
- Average Words per Page: Let's say a standard webpage has 1,000 words.

2. Calculating the Total Connections
   If we have 100 Billion pages, and each page has 1,000 words, our system has to process a grand total of:

- 100 Billion pages x 1,000 words/page = 100 Trillion word-to-URL connections

3. How Much Data (Bytes) is That?
   Every single time a word links to a URL, we need to store a unique identifier for that website (a Document ID). Let's assume a standard 64-bit integer ID, which takes up 8 Bytes of space.

- 100 Trillion connections x 8 Bytes = 800 Terabytes of raw data
  <br>
  If you add metadata like word positions (for exact phrase matching) and PageRank scores, this number easily multiplies by 5 to 10 times, pushing the index size into 4 to 8 Petabytes (4,000 to 8,000 Terabytes).

Can We Fit This in RAM?
<br>
A single, very powerful modern server can hold about 1 to 2 Terabytes of RAM.Our index requires hundreds or thousands of Terabytes.Because 800+ Terabytes cannot physically fit into the memory of a single computer, it is mathematically impossible to run Google Search on one machine.

Now, We Are Ready for Sharding
Now that we have done the estimation, we know why we must shard. We have a 4-Petabyte database that needs to be sliced up and distributed across thousands of separate computers.We can split this pie in two ways:

- Term-Based Sharding: Server 1 holds words A-D, Server 2 holds E-H, etc.
- Document-Based Sharding: Server 1 holds the index for Websites 0-1 Million, Server 2 holds Websites 1 Million to 2 Million, etc.

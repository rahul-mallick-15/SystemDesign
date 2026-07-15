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

```
Term Sharding would mean words like "world cup" and "election" spikes would overload corresponding servers where they are stored
But "document sharding", would allow more distribution since corresponding servers could be in many numbers, but how will the search work here ?

Does every search need to be forwarded to all servers, since the results need to come from all servers ? Or it just needs to be the first server ?
```

Analysis of Term Sharding is exactly why companies like Google avoid it for core search—popular words create massive traffic bottlenecks on specific servers.
<br>
In Document sharding, because the data is split by website ID, the index entries for a word like "election" are scattered across every single server in your cluster. This completely spreads out the traffic load.

<b>How do we find anything without asking everyone?</b>

The Reality: Yes, You Have to Ask Every Single Server
<br>
In a standard Document-Sharded system, a single search query must be sent to all servers (or a large subset of them) simultaneously.
<br>
This process uses a pattern called Scatter-Gather. Here is exactly how it works when a user searches for the word "pizza":

```
                  [ User Types: "pizza" ]
                             │
                             ▼
                    ┌─────────────────┐
                    │  Load Balancer  │
                    └────────┬────────┘
                             │
            ┌────────────────┼────────────────┐
            ▼                ▼                ▼
     ┌────────────┐   ┌────────────┐   ┌────────────┐
     │  Server 1  │   │  Server 2  │   │  Server 3  │  (SCATTER)
     │ (Docs 1-1M)│   │(Docs 1M-2M)│   │(Docs 2M-3M)│
     └──────┬─────┘   └──────┬─────┘   └──────┬─────┘
            │                │                │
            └────────────────┼────────────────┘
                             ▼
                    ┌─────────────────┐
                    │ Aggregator Node │                (GATHER)
                    └────────┬────────┘
                             │ Top 10 Results
                             ▼
                    [ User Sees Results ]
```

- Scatter: A central coordinator server (the Aggregator) receives the query "pizza". It broadcasts this query to all 1,000 servers at the same time.
- Local Search: Each individual server looks through its own tiny index. Server 1 looks up "pizza" only for websites #1 to #1,000,000. It finds its own local top results.
- Gather: Each server sends its top results back to the Aggregator.
- Merge: The Aggregator sorts the incoming results, picks the absolute best 10 pages, translates those 64-bit Document IDs back into real text URLs, and sends them to the user.

<h4> Why Asking Every Server is Actually a Good Thing</h4>
At first glance, hitting 1,000 servers for a single search sounds terrible. However, it is incredibly efficient for two reasons:

- True Parallelism: Instead of one giant database struggling to read a massive list on a slow hard drive, 1,000 cheap computers are reading tiny, bite-sized files simultaneously. The search finishes in a fraction of the time.
- No Single Bottleneck: If "world cup" trends, all 1,000 servers share 1/1,000th of the work. No single machine catches fire.

<h4> The Problem of "The Slowest Link" </h4>
While Document Sharding is great, it introduces a major engineering problem known as the Tail Latency or Long Pole problem.If your search engine has to wait for all 1,000 servers to reply before showing the user a page, your overall search speed is entirely dictated by the slowest single server in the network. If Server #547 gets stuck or encounters a hardware hiccup, the user's search freezes.

<h5>The Roadblock: Why We Can't Just Stop Early </h5>
In a standard database, if you need 10 records, you can stop as soon as you find any 10 records. But Google is a ranked search engine. The user doesn't just want any 10 pages containing the word "pizza"—they want the absolute best 10 pages on the entire internet.If the server holding the absolute #1 best result (like wikipedia.org/wiki/Pizza) happens to be the slow server, and your aggregator cuts off early to return results from faster servers, the user gets a page of low-quality, irrelevant websites.Because we cannot sort the top 10 accurately until we hear from everyone, we cannot easily use a simple "stop-as-soon-as-we-have-10" approach.

<h5> How Google Actually Solves It: The 3 Industry Workarounds </h5>

Instead of stopping early and risking bad results, Google and other major search engines use three specific engineering tactics to beat the "slowest link" problem:

1. <h6>Hedged Requests (The Backup Query)</h6>
   This is a brilliant technique pioneered by Google’s infrastructure teams.The Aggregator sends the search query to Server A.If Server A does not reply within a tiny threshold (say, 15 milliseconds), the Aggregator assumes Server A is having a bad day.It immediately sends the exact same query to a backup replica (Server A-copy).Whichever server replies first wins, and the slower request is cancelled. This prevents a single hardware hiccup from ruining the search speed.

2. <h6> Graceful Degradation (Dropping the Tail) </h6>
   If the backup request also fails or takes too long, and a hard timeout is reached (e.g., 200 milliseconds), the aggregator finally chooses to drop that server's data.It will merge the results from the other 999 servers and show them to the user.Missing 0.1% of the internet for a split second is better than keeping a human being waiting in front of a blank, loading screen.

3. <h6> Caching Popular Queries </h6>
   The easiest way to avoid the slow server problem is to not hit the servers at all.Millions of people search for identical words like "weather" or "pizza" every day.Google saves the pre-compiled top 10 results for these popular keywords in a super-fast Global Cache memory layer.If a search hits the cache, it bypasses the 1,000 document servers entirely.

```
Do you update the index files live (which causes data locking and slow searches), or do you write new, separate index files and merge them later using background workers (like a Log-Structured Merge-tree or MapReduce architecture)?
```

Because our index is stored as highly optimized, sorted lists of numbers (Document IDs), inserting a single new website into the middle of a sorted file forces the computer to rewrite the entire rest of that file on the disk. Doing this billions of times a day is incredibly slow and wears out hardware.
<br>
Instead, systems like Google use Immutability—meaning they never modify an existing file. They only write new ones. This is where Log-Structured Merge-trees (LSM-Trees) and MapReduce come into play.

1. <h4> Log-Structured Merge-tree (LSM-Tree) </h4>
   An LSM-Tree is a database design pattern optimized for incredibly fast writes. Instead of updating a giant file on the disk immediately, it breaks the process into two stages:

- Write to Memory (Fast): When new web pages are indexed, the changes are written to a fast, sorted structure inside the computer’s RAM (called a MemTable). Because it is in memory, this is nearly instant.
- Flush to Disk (Immutable): Once the memory fills up, the system flushes that sorted data down to the hard drive as a permanent, unchangeable file called an SSTable.

<b>The Problem</b>: Too Many Small Files
<br>
Over time, you end up with hundreds of tiny, separate index files on your disk. If a user searches for "pizza", the server has to look inside every single one of those files, which slows down search speeds.

<b>The Solution</b>: Compaction (The "Merge" part)
<br>
In the background, a separate system continually reads these small, sorted files, merges them together into one large, perfectly ordered file, and deletes the old duplicates. This ensures search reads remain lightning fast without ever locking up the live database.

2. <h4>MapReduce Architecture</h4>
   While LSM-Trees handle continuous updates on individual servers, MapReduce is a framework used to process and index the entire internet all at once in massive daily or weekly batches. It was invented by Google to solve the exact problem we are discussing

It breaks a massive data problem down into two simple steps across thousands of computers: Map and Reduce.

<h6>Step 1: Map (Break down the data)</h6>
Thousands of worker computers take raw web pages downloaded by the crawler and break them down into basic key-value pieces.

`Input page: site-A.com contains "pizza recipe"`<br>
`Map Output: ("pizza", site-A.com), ("recipe", site-A.com)`

<h6>Step 2: Shuffle & Sort (Organize behind the scenes)</h6>
The MapReduce framework automatically groups all identical words together from every single mapping computer and sends them to the designated final processor.

<h6>Step 3: Reduce (Glue it together)</h6>
Another set of computers takes all the scattered references to a single word and condenses them into a single, clean list.

`Before Reduce: ("pizza", site-A.com), ("pizza", site-B.com)`<br>
`Reduce Output: "pizza" -> [site-A.com, site-B.com]`<br>
Once the Reduce step finishes, a brand new, perfectly optimized index database file is generated and pushed to the live search servers.

1. The RAM vs. Disk Confusion: When Did We Move to Files?

- The Index Lives in Files on Disk: Your total index is 800+ Terabytes. It must be saved permanently as files on hard drives (SSD/Disk) because if a server loses power, anything stored only in RAM vanishes instantly.
- The Index is Loaded into RAM for Searching: When the 1,000 servers boot up, they open those database files and load as much of the index data as possible directly into their RAM.

  When a user searches, the server reads the data from its ultra-fast RAM. The disk is only used as a permanent safety net to hold the files and reload them if a server crashes.

2. What About the 1-2 TB RAM x 1,000 Servers?
   If we have 1,000 servers, and each has 1 to 2 Terabytes of RAM, our cluster has a total combined RAM capacity of 1,000 to 2,000 Terabytes (1 to 2 Petabytes).Our calculated index size was 800 Terabytes.Because 800 Terabytes < 2,000 Terabytes, the entire global index can fit entirely inside the combined RAM of our 1,000 document sharded servers. Each individual server only needs to hold about 800 Gigabytes of data in its memory.

3. How is Shuffle & Sort Different From Reduce?
   Think of MapReduce like a giant mail sorting facility.

- The Map Step: Workers look at letters (webpages) and write down the destination zip code (the word).
- The Shuffle & Sort Step (The Transportation): This is an automated logistics phase. The system takes all letters marked "90210" from 500 different mapping computers, groups them together, sorts them alphabetically, and physically routes them across the network to the one specific computer handling the "90210" area code. No data is changed here; it is just being moved and organized.
- The Reduce Step (The Final Math): The destination computer takes that giant stack of routed letters, opens them, extracts the final content, and compresses them into a single, clean master list.

4. LSM-Trees: Is it Done on Disk?
   Mostly. An LSM-tree keeps a tiny fraction of data (the newest updates) in RAM. But 99% of its data is stored on the disk as unchangeable files called SSTables.
   When a search happens, the server checks the RAM first (for the newest updates) and then checks its RAM-cached copies of the disk files.

5. The Lock Problem: Does Merging Block the Reads?
   This is the beauty of Immutability. Because the database files are unchangeable, reads are never blocked, and the database is never locked.Here is exactly how the background system merges files without stopping live searches:

- The Search State: Server 1 is actively reading from two old files on the disk: File_A and File_B.- The Background Merge: A separate background worker thread reads File_A and File_B, merges them together, and writes a brand new file called File_C.
- Zero Interruption: While File_C is being written, live users are still searching through File_A and File_B perfectly fine.
- The Atomic Switch: The moment File_C is 100% complete, the system flips a tiny pointer in memory. New searches are instantly told to look at File_C.
- Clean Up: The system safely deletes the old File_A and File_B because no one is looking at them anymore.

  By creating a new file instead of modifying an old one, Google can update its index continuously without a single microsecond of search downtime.

<h4> The Disconnect: Processing vs. Serving </h4>
To build an index efficiently, you must group by word. It is impossible to build an Inverted Index without a step where all data for the word "pizza" goes to one place to be compiled.
<br>
MapReduce does exactly that. The computer handling the "90210" word outputs a single, giant file containing the master list of all Document IDs for that word. This output file is Term-Sharded (organized by word).
<br>
If we stopped there and pushed those files directly to our search cluster, we would have a Term-Sharded system—and we already proved that popular words would crash those servers.

<h4> The Final Step: The Great Redistribution </h4>
To get the best of both worlds, Google takes the Term-Sharded files generated by MapReduce and runs a final, fast slicing step before deploying them to the 1,000 live search servers.
<br>
Imagine the MapReduce computer finishes compiling the master list for the word "pizza":<br>
"pizza" → [Doc 12, Doc 5,500, Doc 1,200,500, Doc 3,500,000]

Instead of keeping this list together on one machine, the system slices this list up based on the Document IDs:

- Doc IDs 1 to 1 Million belong to Server 1. So, Doc 12 and Doc 5,500 are sent to Server 1.
- Doc IDs 1M to 2 Million belong to Server 2. So, Doc 1,200,500 is sent to Server 2.
- Doc IDs 3M to 4 Million belong to Server 4. So, Doc 3,500,000 is sent to Server 4.

```
[ MapReduce Output (Term-Sharded) ]
             │
             ▼
      "pizza" ──> [ Doc 12, Doc 5500, Doc 1.2M, Doc 3.5M ]
                        │         │        │         │
       ┌────────────────┘         │        │         └──────────────┐
       ▼                          ▼        ▼                        ▼
┌──────────────┐          ┌──────────────┐                  ┌──────────────┐
│   Server 1   │          │   Server 2   │                  │   Server 4   │
│ (Docs 1-1M)  │          │ (Docs 1M-2M) │                  │ (Docs 3M-4M) │
├──────────────┤          ├──────────────┤                  ├──────────────┤
│ "pizza":     │          │ "pizza":     │                  │ "pizza":     │
│ [12, 5500]   │          │ [1,200,500]  │                  │ [3,500,000]  │
└──────────────┘          └──────────────┘                  └──────────────┘
                    [ Final Serving Layer (Document-Sharded) ]
```

<b>The Result</b>
<br>
When the redistribution is complete, every single one of the 1,000 live search servers ends up with its own tiny local "pizza" list.

- During Processing (MapReduce): The data is temporarily grouped by Word because that is the only logical way to aggregate the lists.
- During Serving (Live Search): The data is cut up and stored by Document ID because that is the only stable way to handle billions of fast queries without overloading the hardware.

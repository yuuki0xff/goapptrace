# Goapptrace SQL Interface Specification
Goapptraceでは、SQLを使ってログを検索するインターフェースを持つ。


## Interface
### CLI
```
$ goapptrace log query --format csv [--follow] {LogID} {SQL}
```

NOTE: `--format`オプションと`--follow`オプションは、現時点では未実装。


### REST API
```
GET /log/{log-id}/search.csv?sql={SQL}
```



## SQL Specification
### SQL Examples
```
SELECT * FROM calls WHERE gid=0;
SELECT * FROM calls WHERE FRAME(module like 'main.%');
SELECT * FROM calls WHERE starttime > DATE_SUB(NOW(), INTERVAL 1 MINUTE);
SELECT * FROM frames GROUP BY file, line ORDER BY COUNT(1);
SELECT * FROM goroutines WHERE exectime > '1s';
```


## Table Definitions
```
CREATE TABLE calls (
	id BIGINT PRIMARY KEY,
	gid BIGINT,
	starttime DATETIME,
	endtime DATETIME,
	exectime BIGINT,
);
CREATE TABLE frames (
	id BIGINT,
	offset BIGINT,
	module TEXT,
	func TEXT,
	file TEXT,
	line INT,
	pc BIGINT,
	PRIMARY KEY (id, offset)
);
CREATE TABLE goroutines (
	gid BIGINT PRIMARY KEY,
	starttime DATETIME,
	endtime DATETIME,
	exectime BIGINT
);
CREATE TABLE funcs (
	name TEXT PRIMARY KEY,
	shortname TEXT,
	package TEXT,
	path TEXT
);
CREATE TABLE modules (
	module TEXT PRIMARY KEY
);
```


## Functions
```
FRAME(expr)
- alias of EXISTS(SELECT 1 FROM frames WHERE (frames.id = calls.id) AND (expr))
CALL(expr)
- alias of EXISTS(SELECT 1 FROM calls WHERE (calls.id = frames.id) AND (expr))
```


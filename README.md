## pgcacher 

`pgcacher` is used to get page cache statistics for files. Use the **pgcacher** command to know how much cache space the fd of the specified process occupies in the page cache.  Use **pgcacher** to know whether the specified file list is cached in the page cache, and how much space is cached.

Compared with pcstat, `pgcacher` has fixed the problem that the file list of the process is incorrect. It used to be obtained through `/proc/{pid}/maps`, but now it is changed to obtain from `/proc/{pid}/maps` and `/proc/{pid}/fd` at the same time. pgcacher supports more parameters, such as top, worker, least-size, exclude-files and include-files. 😁

In addition, the pgcacher code is more robust, and also supports concurrency parameters, which can calculate the cache occupancy in the page cache faster.

> the some code of pgcacher copy from pcstat and hcache.

## Usage

```sh
pgcacher <-json <-pps>|-terse|-default> <-nohdr> <-bname> file file file
    -worker concurrency workers, default: 2
    -pid show all open maps for the given pid
    -top show top x cached files in descending order
    -lease-size ignore files smaller than the lastSize, such as '10MB' and '15GB'
    -exclude-files exclude the specified files by wildcard, such as 'a*c?d' and '*xiaorui*,rfyiamcool'
    -include-files only include the specified files by wildcard, such as 'a*c?d' and '*xiaorui?cc,rfyiamcool'
    -json output will be JSON
    -pps include the per-page information in the output (can be huge!)
    -terse print terse machine-parseable output
    -histo print a histogram using unicode block characters
    -nohdr don't print the column header in terse or default format
    -bname use basename(file) in the output (use for long paths)
    -plain return data with no box characters
    -unicode return data with unicode box characters
```

## Install

**compile source code**

```sh
git clone https://github.com/rfyiamcool/pgcacher.git
cd pgcacher
make build
sudo cp pgcacher /usr/local/bin/ 
```

**use binary directly**

test pass on ubuntu, centos 7.x and centos 8.x.

```
wget xiaorui-cc.oss-cn-hangzhou.aliyuncs.com/files/pgcacher
chmod 777 pgcacher
\mv pgcacher /usr/local/bin
```

## Usage

```
$ sudo pgcacher -pid=29260 -worker=5
+-------------------+----------------+-------------+----------------+-------------+---------+
| Name              | Size           │ Pages       │ Cached Size    │ Cached Pages│ Percent │
|-------------------+----------------+-------------+----------------+-------------+---------|
| /root/rui/file4g  | 3.906G         | 1024000     | 3.906G         | 1024000     | 100.000 |
| /root/rui/file3g  | 2.930G         | 768000      | 2.930G         | 768000      | 100.000 |
| /root/rui/file2g  | 1.953G         | 512000      | 1.953G         | 512000      | 100.000 |
| /root/rui/file1g  | 1000.000M      | 256000      | 1000.000M      | 256000      | 100.000 |
| /root/rui/open_re | 1.791M         | 459         | 1.791M         | 459         | 100.000 |
|-------------------+----------------+-------------+----------------+-------------+---------|
│ Sum               │ 9.767G         │ 2560459     │ 9.767G         │ 2560459     │ 100.000 │
+-------------------+----------------+-------------+----------------+-------------+---------+

$ dd if=/dev/urandom of=file1g bs=1M count=1000
$ dd if=/dev/urandom of=file2g bs=1M count=2000
$ dd if=/dev/urandom of=file3g bs=1M count=3000
$ dd if=/dev/urandom of=file4g bs=1M count=4000
$ cat file1g file2g file3g file4g > /dev/null

$ sudo pgcacher file1g file2g file3g file4g
+--------+----------------+-------------+----------------+-------------+---------+
| Name   | Size           │ Pages       │ Cached Size    │ Cached Pages│ Percent │
|--------+----------------+-------------+----------------+-------------+---------|
| file4g | 3.906G         | 1024000     | 3.906G         | 1024000     | 100.000 |
| file3g | 2.930G         | 768000      | 2.930G         | 768000      | 100.000 |
| file2g | 1.953G         | 512000      | 1.953G         | 512000      | 100.000 |
| file1g | 1000.000M      | 256000      | 1000.000M      | 256000      | 100.000 |
|--------+----------------+-------------+----------------+-------------+---------|
│ Sum    │ 9.766G         │ 2560000     │ 9.766G         │ 2560000     │ 100.000 │
+--------+----------------+-------------+----------------+-------------+---------+

$ sudo pgcacher /root/rui/*
+------------+----------------+-------------+----------------+-------------+---------+
| Name       | Size           │ Pages       │ Cached Size    │ Cached Pages│ Percent │
|------------+----------------+-------------+----------------+-------------+---------|
| file4g     | 3.906G         | 1024000     | 3.906G         | 1024000     | 100.000 |
| file3g     | 2.930G         | 768000      | 2.930G         | 768000      | 100.000 |
| file2g     | 1.953G         | 512000      | 1.953G         | 512000      | 100.000 |
| testfile   | 1000.000M      | 256000      | 1000.000M      | 256000      | 100.000 |
| file1g     | 1000.000M      | 256000      | 1000.000M      | 256000      | 100.000 |
| pgcacher   | 2.440M         | 625         | 2.440M         | 625         | 100.000 |
| open_re    | 1.791M         | 459         | 1.791M         | 459         | 100.000 |
| cache.go   | 19.576K        | 5           | 19.576K        | 5           | 100.000 |
| open_re.go | 644B           | 1           | 644B           | 1           | 100.000 |
| nohup.out  | 957B           | 1           | 957B           | 1           | 100.000 |
|------------+----------------+-------------+----------------+-------------+---------|
│ Sum        │ 10.746G        │ 2817091     │ 10.746G        │ 2817091     │ 100.000 │
+------------+----------------+-------------+----------------+-------------+---------+
```

## pgcacher design

![](https://xiaorui-cc.oss-cn-hangzhou.aliyuncs.com/images/202303/202303121052113.png)

![](https://xiaorui-cc.oss-cn-hangzhou.aliyuncs.com/images/202303/202303131739063.png)


## Thanks to

@tobert for pcstat

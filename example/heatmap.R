library(dplyr)

# goapptrace.query executes a specified SQL and returns the result.
goapptrace.query <- function(HostPort, LogID, SQL){
    URL <- paste("http://", HostPort, "/api/v0.1/log/", LogID, "/search.csv?sql=", URLencode(SQL), sep = "")
    X <- read.csv(url(URL), header = T)
    return(X)
}

# goapptrace.exectimePerGoroutine returns a crosstab for execution time per goroutine.
# Usage:
#   source("example/heatmap.R")
#   LogID <- "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
#   X <- goapptrace.exectimePerGoroutine(LogID)
#   heatmap(X, Colv = NA, Rowv = NA)
goapptrace.exectimePerGoroutine <- function(LogID) {
    X <- goapptrace.query("localhost:8700", LogID, 'SELECT gid, exectime FROM calls WHERE id < 1000000')
    X1 <- data.frame(X)
    X2 <- X1 %>%
        filter(exectime > 0) %>%
        group_by(exectime_log10 = floor(log10(exectime))) %>%
        arrange(desc(exectime_log10))
    X3 <- table(unlist(X2["exectime_log10"]), unlist(X2["gid"]))
    rownames(X3) <- paste("10^", rownames(X3), " ns", sep="")
    return(X3)
}


// Initialization sequence
//   1. Execute other `init()` functions. Generated logs are buffered into `logger.initBuffer` array.
//   2. Execute `logger.init()`. It is initialize the `output` variable, and sends buffered logs on `initBuffer`.
//   3. Execute other `init()` functions.
//   4. Start the main routine.
package logger

#include <include/CommandHandler.h>

#include <vector>
#include <sstream>
#include <algorithm>

std::vector<std::string> CommandHandler::parseCommand(const std::string &input) {
    std::vector<std::string> tokens;
    if (input.empty()) return tokens;

    // If it doesnt strart with '*', fallback to splitting by whitespace.
    if (input[0] != '*') {
        std::istringstream iss(input);
        std::string token;
        while (iss >> token)
            tokens.push_back(token);
        return tokens;
    }

    size_t pos = 0;
    // Expect '*' followed by number of elements
    if (input[pos] != '*') return tokens;
    pos++; // skip '*'

    // crlf = Carriage Return (\r), Line Feed (\n)
    size_t crlf = input.find("\r\n", pos);
    if (crlf == std::string::npos) return tokens;

    int numElements = std::stoi(input.substr(pos, crlf - pos));
    pos = crlf + 2;

    for (int i = 0; i < numElements; i++) {
        if (pos >= input.size() || input[pos] != '$') break; // format error
        pos++; // skip '$'

        crlf = input.find("\r\n", pos);
        if (crlf == std::string::npos) break;
        int len = std::stoi(input.substr(pos, crlf - pos));
        pos = crlf + 2;

        if (pos + len > input.size()) break;
        std::string token = input.substr(pos, len);
        tokens.push_back(token);
        pos += len + 2; // skip token and CRLF
    }
    return tokens;
}


std::string CommandHandler::processCommand(const std::string& commandLine) {
    // RESP parser
    std::vector<std::string> commands = parseCommand(commandLine);
    if(commands.empty()) return "Error: Empty command recieved\n";

    std::string cmd = commands[0];
    std::transform(cmd.begin(), cmd.end(), cmd.begin(), ::toupper);

}
# Go Classroom Microphone System

A web-based classroom system where up to 200 students can use their mobile phones as microphones. Students can raise their hands, and teachers can allow one student at a time to speak.

## Features

- **Student Interface**: Students enter their name and select a channel (1-200)
- **Teacher Interface**: Teachers can see all students with their names and channel numbers
- **Hand Raising**: Students can raise their hand to request speaking
- **Single Speaker**: Only one student can speak at a time (teacher can switch between students)
- **TURN Server Support**: Works across different networks (mobile data, different WiFi networks)
- **Real-time Communication**: WebSocket for signaling, WebRTC for audio streaming

## Setup

1. Install dependencies:
   ```bash
   go mod tidy
   ```

2. Create `.env` file with TURN server credentials:
   ```
   PORT=9000
   TURN_URL=turn:TURNSERVERIP:3478
   TURN_USERNAME=
   TURN_CREDENTIAL=
   ```

3. Run the server:
   ```bash
   go run main.go
   ```

4. Open in browser:
   - Teacher: `http://localhost:9000/teacher.html`
   - Student: `http://localhost:9000/`

## Usage

### For Students:
1. Enter your name
2. Select a channel number (1-200)
3. Click "Connect to Classroom"
4. Click "Raise Hand" when you want to speak
5. Wait for teacher to allow you to speak
6. Click "Speak" when allowed

### For Teachers:
1. Open the teacher interface
2. See students who have raised their hands
3. Click "Allow to Speak" to let a student speak
4. Click "Mute" to stop a student from speaking
5. Only one student can speak at a time

## Technical Details

- **Backend**: Go with Echo framework and Gorilla WebSocket
- **Frontend**: HTML5, CSS3, JavaScript with WebRTC
- **Audio**: WebRTC getUserMedia for microphone access
- **Signaling**: WebSocket for real-time communication
- **NAT Traversal**: TURN server for cross-network connectivity
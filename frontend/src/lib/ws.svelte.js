let socket;
let msgs = $state([]); 

export function connect_ws(group_name, onMessageCallback) {
	socket = new WebSocket(`ws://127.0.0.1:3000/ws?room=${group_name}`);

	socket.onopen = () => {
		console.log("Connected to WebSocket");
	};

	socket.onmessage = (event) => {
		const data = event.data;
		msgs = [...msgs, data];
		if (onMessageCallback) {
			onMessageCallback(data);
		}
	};

	socket.onclose = () => {
		console.log("Disconnected from WebSocket");
	};

	socket.onerror = (err) => {
		console.error("WebSocket error:", err);
	};
}

export function send_msg(msg) {
	if (socket && socket.readyState === WebSocket.OPEN) {
		socket.send(msg);
	}
}

export function get_msgs() {
	return msgs;
}

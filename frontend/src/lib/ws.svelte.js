let socket
let msgs = $state([])

export function connect_ws(group_name) {
    socket = new WebSocket(`ws://127.0.0.1:3000/ws?room=${group_name}`)

    socket.onopen = () => {
        console.log("connected to websocket")
    }

    socket.onmessage = (event) => {
        console.log(event.data)
        msgs = [...msgs, event.data]
    }

    socket.onclose = () => {
        console.log("disconnected from websocket")
    }

    socket.onerror = (err) => {
        console.error("error websocket: ", err)
    }
}

export function send_msg(msg) {
    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(msg)
    }
}

export function get_msgs() {
    return msgs
}

(function() {
    var socket = new WebSocket("ws://localhost:35729/ws");

    socket.onopen = function() {
        console.log("Livereload connected");
    };

    socket.onmessage = function(event) {
        if (event.data === "reload") {
            console.log("Reloading...");
            window.location.reload();
        }
    };

    socket.onclose = function() {
        console.log("Livereload disconnected");
    };

    socket.onerror = function(error) {
        console.error("Livereload error: " + error);
    };
})();

window.onload = function() {
    refreshWorkers();
};

Vue.use(VeeValidate);

Vue.component('worker-progress', {
    props:['status'],
    template:'#worker-progress'
});

Vue.component('worker-item', {
    props: ['worker'],
    template:'#worker-item'
});

var workerlist = new Vue({
    el: '#app',
    data: {
        workers: [],
        accept: '',
        url: '',
        location: ''
    },
    methods: {
        addDownload(){
            this.$validator.validateAll();

            if (!this.errors.any()) {
                fetch('/api/tasks',{
                    method: "POST",
                    body: JSON.stringify({
                        url:this.url,
                        accept:this.accept,
                        location:this.location
                    })
                }).then((response) => {
                    if(response.ok){
                        this.workers.forEach((w) => {
                            w.ws.close();
                        });
                        refreshWorkers();
                    }
                });
                console.log(this.url);
                console.log(this.accept);
                console.log(this.location);
            }
        }
    }
});


function refreshWorkers() {
    fetch('/api/workers')
        .then((response) => response.json())
        .then((workers) => {
            workers.sort((a, b) => {
                if (a.Filename < b.Filename) {
                    return -1;
                } else {
                    return 1;
                }
            });
            workers.forEach((w) => {
                var loc = window.location;
                var uri = 'ws:';

                if (loc.protocol === 'https:') {
                    uri = 'wss:';
                }
                uri += '//' + loc.host;
                uri += '/api/workers/';
                uri += w.Id + '/';
                uri += 'ws';

                fetch(`/api/workers/${w.Id}/active`)
                    .then((response) => {
                        if (!response.ok) {
                            throw Error(response.statusText);
                        }
                        return response;
                    })
                    .then(() => {
                        w.ws = new WebSocket(uri);
                        w.ws.onmessage = (evt) => {
                            var status = JSON.parse(evt.data);
                            w.Status = status;
                        };
                    })
                    .catch(() => {
                        console.log("ignoring" + w.Id);
                    });

            });
            workerlist.workers = workers;
        });
}

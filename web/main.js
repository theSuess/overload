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
                });
            }
        },
        refreshWorkers() {
            fetch('/api/workers')
                .then((response) => response.json())
                .then((workers) => {
                    if(workers == null) {
                        window.setTimeout(this.refreshWorkers,5000); // Try again in 5 Seconds
                        return;
                    }
                    workers.sort((a, b) => {
                        if (a.filename < b.filename) {
                            return -1;
                        } else {
                            return 1;
                        }
                    });
                    workers.forEach((w) => {
                        w.status.isFinished = () => w.status.total === w.status.length;
                    });
                    this.workers = workers;
                });
        }
    }
});

workerlist.refreshWorkers();

var loc = window.location;
var uri = '';
if (loc.protocol === 'https:') {
    uri = 'wss:';
} else {
    uri = 'ws:';
}
uri += '//' + loc.host;
uri += '/api/status';
var ws = new WebSocket(uri);

ws.onmessage = (evt) => {
    var status = JSON.parse(evt.data);
    var workeritem = workerlist.$refs[`worker-${status.workerid}`];
    if(workeritem != undefined) {
        status.isFinished = () => {
            if(status.total === status.length) {
                return true;
            } else {
                return false;
            }
        };
        workeritem[0].worker.status=status;
    }
};

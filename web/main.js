Vue.use(VeeValidate);

Vue.component('worker-progress', {
    props:['status'],
    template:'#worker-progress'
});

Vue.component('worker-item', {
    props: ['worker'],
    template:'#worker-item'
});
const NotificationStore = {
    state: [], // here the notifications will be added

    addNotification: function (notification) {
        this.state.push(notification);
    },
    removeNotification: function (notification) {
        var index = this.state.indexOf(notification);
        this.state.splice(index,1);
    }
};

var Notification = Vue.extend({
	  template: '#notification',
    props: ['notification'],
    data: function () {
  	    return { timer: null };
	  },
    mounted: function () {
        let timeout = this.notification.hasOwnProperty('timeout') ? this.notification.timeout : true;
        if (timeout) {
  	        let delay = this.notification.delay || 3000;
            this.timer = window.setTimeout(function () {
                this.triggerClose(this.notification);
            }.bind(this), delay);
        }
    },

    methods: {
        triggerClose: function (notification) {
    	      clearTimeout(this.timer);
            NotificationStore.removeNotification(notification);
        }
    }
});
var Notifications = Vue.extend({
	  template: '#notifications',
    components: {
        notification: Notification
    },
    data () {
        return {
            notifications: NotificationStore.state
        };
    }
});

var app = new Vue({
    el: '#app',
    components: {
  	    'notifications': Notifications
    },
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
        },
        notifyDone(filename){
            NotificationStore.addNotification({
      	        title: 'Download Finished',
                text: `${filename} has finished downloading`,
                type: "is-primary",
                timeout: true,
                delay: 10000
            });
        }
    }
});

app.refreshWorkers();

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
    var workeritem = app.$refs[`worker-${status.workerid}`];
    if(workeritem != undefined) {
        status.isFinished = () => {
            if(status.total === status.length) {
                app.notifyDone(workeritem[0].worker.filename);
                return true;
            } else {
                return false;
            }
        };
        workeritem[0].worker.status=status;
    }
};

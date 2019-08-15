new Vue({
  el: '#app',
  data: function() {
    return {
      dialogFormVisible: false,
      dialogConfirmVisible: false,
      tableData: [],
      formLabelWidth: '120px',
      deleteLocal: '',
      api: 'http://127.0.0.1:3001',
      isAdd: false,
      form: {
        target: '',
        local: ''
      },
      loading: true
    };
  },
  methods: {
    handleFormAction() {
      if (this.isAdd) {
        this.handleAddAction();
      } else {
      }
    },
    handleEdit(local, target) {
      this.form.local = local;
      this.form.target = target;
      this.isAdd = false;
      this.dialogFormVisible = true;
    },
    handleDelete(local) {
      this.deleteItem(local);
    },
    handleAdd() {
      this.form.local = '';
      this.form.target = '';
      this.isAdd = true;
      this.dialogFormVisible = true;
    },
    handleReload() {
      this.loadList();
    },
    handleAddAction() {
      this.dialogFormVisible = false;
      this.addItem(this.form.local, this.form.target);
    },
    handleUpdateAction() {
      this.dialogFormVisible = false;
      this.updateItem(this.form.local, this.form.target);
    },
    async loadList() {
      this.loading = true;
      const resp = await (await fetch(`${this.api}/proxy`, { mode: 'cors' })).json();
      const keys = Object.keys(resp);
      this.tableData = keys.map(key => {
        return {
          local: key,
          target: resp[key]
        };
      });
      this.loading = false
    },
    async addItem(local, target) {
      this.loading = true;
      await fetch(`${this.api}/proxy`, {
        method: 'POST',
        body: JSON.stringify({ local, target }),
        mode: 'cors'
      });
      this.loadList();
    },
    async updateItem(local, target) {
      this.loading = true;
      await fetch(`${this.api}/proxy/${encodeURIComponent(local)}`, {
        method: 'PATCH',
        body: JSON.stringify({ target }),
        mode: 'cors'
      });
      this.loadList();
    },
    async deleteItem(local) {
      this.loading = true;
      console.warn(this);
      await fetch(`${this.api}/proxy/${encodeURIComponent(local)}`, {
        method: 'DELETE',
        mode: 'cors'
      });
      this.loadList();
    }
  },
  mounted: function() {
    this.loadList();
  }
});

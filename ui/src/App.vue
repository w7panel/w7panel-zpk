<template>
	<div style="min-height:99%;">
		<router-view :userInfo="userInfo"></router-view>
	</div>
</template>

<script>
import myAxios from "./utils";
export default {
	data() {
		return { userInfo: {} }
	},
	async created() {
		window.$wujie?.bus.$on("routeChange", (path) => {
			this.$router.push(path)
		});

		await this.getUserInfo()
	},
	methods: {
		getUserInfo() {
			myAxios.get("/v2/api/user/cur-user/info").then(res => {
				this.userInfo = res.data?.data?.user || {}
			});
		}
	}
}
</script>

<style>
.create-item-menupopover {
	padding: 0 !important;
}

.cursor {
	cursor: pointer;
}

.content {
	padding: 20px;
}

.arco-btn+.arco-btn {
	margin-left: 8px;
}

.arco-btn-group .arco-btn+.arco-btn {
	margin-left: 0;
}

.arco-pagination-options {
	order: -1;
	margin-right: 8px;
	margin-left: 0 !important;
}

.arco-table {
	color: #1d2129;
}

.arco-table .arco-table-container,
.arco-table-border .arco-table-container,
.arco-table-border:not(.arco-table-border-cell) .arco-table-container {
	border: 0 !important;
}

.arco-table .arco-table-th,
.table-header .arco-table-th {
	color: #1d2129 !important;
	font-weight: 400 !important;
	background: #f2f3f5 !important;
	border-bottom: 1px solid #f2f3f5 !important;
}

.arco-table .arco-table-td {
	color: #1d2129;
	background: transparent;
	border-bottom: 1px solid #f2f3f5 !important;
}

.arco-table .arco-table-cell {
	min-height: 22px;
	padding: 10px 16px !important;
	line-height: 22px;
}

.arco-table-hover:not(.arco-table-dragging) .arco-table-tr:not(.arco-table-tr-empty):not(.arco-table-tr-summary):hover .arco-table-td:not(.arco-table-col-fixed-left):not(.arco-table-col-fixed-right),
.arco-table-hover .arco-table-tr-drag .arco-table-td:not(.arco-table-col-fixed-left):not(.arco-table-col-fixed-right) {
	background: #f7f8fa;
}

.arco-table .arco-table-td .arco-btn-text {
	height: auto;
	padding: 0 !important;
	color: #165dff !important;
	line-height: 22px;
	background: transparent !important;
	border: 0 !important;
}

.arco-table .arco-table-td .arco-btn-text:hover,
.arco-table .arco-table-td .arco-btn-text:focus {
	color: #165dff !important;
	background: transparent !important;
}

.arco-table .arco-table-td .arco-btn-text+.arco-btn-text {
	margin-left: 16px;
}

.cluster-box {
	padding: 10px 20px;
	height: 60px;
}

.refresh {
	margin: 0 16px;
	cursor: pointer;
}

.com-table-header {
	background-color: #F2F4F5;
}

@-webkit-keyframes rotation {
	from {
		-webkit-transform: rotate(0deg);
		transform: rotate(0deg);
	}

	to {
		-webkit-transform: rotate(360deg);
		transform: rotate(360deg);
	}
}

@keyframes rotation {
	from {
		-webkit-transform: rotate(0deg);
		transform: rotate(0deg);
	}

	to {
		-webkit-transform: rotate(360deg);
		transform: rotate(360deg);
	}
}

.my-rotate {
	animation: rotation 2s linear infinite;
	-moz-animation: rotation 2s linear infinite;
	-webkit-animation: rotation 2s linear infinite;
	-o-animation: rotation 2s linear infinite;
}

</style>

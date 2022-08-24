import { CommonModule } from '@angular/common';
import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { ButtonComponent } from '../button/button.component';

@Component({
	selector: 'convoy-copy-button',
	standalone: true,
	imports: [CommonModule, ButtonComponent],
	templateUrl: './copy-button.component.html',
	styleUrls: ['./copy-button.component.scss']
})
export class CopyButtonComponent implements OnInit {
	@Input('text') textToCopy!: string;
	@Input('size') size: 'sm' | 'md' = 'sm';
	@Input('className') class!: string;
	@Output('copyText') copy = new EventEmitter();
	textCopied = false;

	constructor() {}

	ngOnInit(): void {}

	copyText() {
		if (!this.textToCopy) return;
		const text = this.textToCopy;
		const el = document.createElement('textarea');
		el.value = text;
		document.body.appendChild(el);
		el.select();
		document.execCommand('copy');
		this.textCopied = true;
		this.copy.emit();
		setTimeout(() => {
			this.textCopied = false;
		}, 3000);
		document.body.removeChild(el);
	}
}